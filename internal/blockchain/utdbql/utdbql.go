// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utdbql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/kaleido-io/firefly/internal/blockchain"
	"github.com/kaleido-io/firefly/internal/config"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"

	_ "modernc.org/ql/driver"
)

type UTDBQL struct {
	ctx          context.Context
	capabilities *blockchain.Capabilities
	events       blockchain.Events
	db           *sql.DB
	eventStream  chan *utEvent
	closed       bool
}

type utDBQLEventType string

const (
	utDBQLEventTypeBroadcastBatch utDBQLEventType = "BroadcastBatch"
	utDBQLEventTypeMined          utDBQLEventType = "TransactionMined"

	eventQueueLength = 50
)

type utEvent struct {
	txType     utDBQLEventType
	identity   string
	trackingID string
	txID       string
	data       []byte
}

func (u *UTDBQL) Name() string {
	return "utdbql"
}

func (u *UTDBQL) Init(ctx context.Context, prefix config.ConfigPrefix, events blockchain.Events) (err error) {

	u.ctx = ctx
	u.capabilities = &blockchain.Capabilities{
		GlobalSequencer: true, // fake for unit testing
	}
	u.events = events
	u.eventStream = make(chan *utEvent, eventQueueLength)

	u.db, err = sql.Open("ql", prefix.GetString(UTDBQLConfURL))
	var tx *sql.Tx
	if err == nil {
		tx, err = u.db.Begin()
	}
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		_, err = tx.Exec("CREATE TABLE IF NOT EXISTS dbqltx ( author string, tracking string, type string, data string );")
	}
	if err == nil {
		_, err = tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS dbqltx_primary ON dbqltx(tracking);")
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		return i18n.WrapError(ctx, err, i18n.MsgDBInitFailed)
	}

	return nil
}

func (u *UTDBQL) Capabilities() *blockchain.Capabilities {
	return u.capabilities
}

func (u *UTDBQL) Start() error {
	go u.eventLoop()
	return nil
}

func (u *UTDBQL) VerifyIdentitySyntax(ctx context.Context, identity string) (string, error) {
	if err := fftypes.ValidateFFNameField(ctx, identity, "identity"); err != nil {
		return "", err
	}
	return identity, nil
}

func (u *UTDBQL) SubmitBroadcastBatch(ctx context.Context, identity string, batch *blockchain.BroadcastBatch) (txTrackingID string, err error) {
	trackingID := fftypes.NewUUID().String()
	b, _ := json.Marshal(&batch)

	var tx *sql.Tx
	var res sql.Result
	if err == nil {
		tx, err = u.db.Begin()
	}
	if err == nil {
		defer func() { _ = tx.Rollback() }()
		res, err = tx.Exec("INSERT INTO dbqltx (author, tracking, type, data) VALUES ($1, $2, $3, $4)", identity, trackingID, utDBQLEventTypeBroadcastBatch, string(b))
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		return "", err
	}
	lid, _ := res.LastInsertId()
	txID := strconv.FormatInt(lid, 10)
	u.eventStream <- &utEvent{
		txType:     utDBQLEventTypeBroadcastBatch,
		identity:   identity,
		trackingID: trackingID,
		txID:       txID,
		data:       b,
	}
	u.eventStream <- &utEvent{
		txType:     utDBQLEventTypeMined,
		identity:   identity,
		trackingID: trackingID,
		txID:       txID,
		data:       nil,
	}
	return trackingID, nil
}

func (u *UTDBQL) eventLoop() {
	for {
		select {
		case <-u.ctx.Done():
			log.L(u.ctx).Debugf("Exiting event loop")
			return
		case ev, ok := <-u.eventStream:
			if !ok {
				return
			}
			log.L(u.ctx).Debugf("Dispatching '%s' event '%s'", ev.txType, ev.txID)
			u.dispatchEvent(ev)
		}
	}
}

func (u *UTDBQL) dispatchEvent(ev *utEvent) {
	var err error
	switch ev.txType {
	case utDBQLEventTypeBroadcastBatch:
		batch := &blockchain.BroadcastBatch{}
		if err := json.Unmarshal(ev.data, batch); err != nil {
			log.L(u.ctx).Errorf("Failed to unmarshal '%s' event '%s': %s", ev.txType, ev.txID, err)
			return
		}
		err = u.events.SequencedBroadcastBatch(batch, ev.identity, ev.trackingID, nil)
	case utDBQLEventTypeMined:
		err = u.events.TransactionUpdate(ev.trackingID, fftypes.TransactionStatusConfirmed, ev.txID, "", nil)
	}
	if err != nil {
		log.L(u.ctx).Errorf("Exiting due to error")
		u.Close()
	}

}

func (u *UTDBQL) Close() {
	if !u.closed {
		close(u.eventStream)
		u.closed = true
		_ = u.db.Close()
	}
}
