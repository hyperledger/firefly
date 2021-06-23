// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
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

package broadcast

import (
	"context"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/log"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInput) (out *fftypes.Message, err error) {
	in.Header.Namespace = ns
	in.Header.Type = fftypes.MessageTypeBroadcast
	if in.Header.Author == "" {
		in.Header.Author = config.GetString(config.OrgIdentity)
	}
	if in.Header.TxType == "" {
		in.Header.TxType = fftypes.TransactionTypeBatchPin
	}

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin
	var dataToPublish []*fftypes.DataAndBlob
	err = bm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
		in.Message.Data, dataToPublish, err = bm.data.ResolveInputDataBroadcast(ctx, ns, in.InputData)
		if err != nil {
			return err
		}

		// If we have data to publish, we break out of the DB transaction, do the publishes, then
		// do the send later - as that could take a long time (multiple seconds) depending on the size
		if len(dataToPublish) > 0 {
			return nil
		}

		return bm.broadcastMessageCommon(ctx, &in.Message)
	})
	if err != nil {
		return nil, err
	}

	// Perform deferred processing
	if len(dataToPublish) > 0 {
		return bm.publishBlobsAndSend(ctx, &in.Message, dataToPublish)
	}

	// The broadcastMessage function modifies the input message to create all the refs
	return &in.Message, err
}

func (bm *broadcastManager) publishBlobsAndSend(ctx context.Context, msg *fftypes.Message, dataToPublish []*fftypes.DataAndBlob) (*fftypes.Message, error) {

	for _, d := range dataToPublish {

		// Stream from the local data exchange ...
		reader, err := bm.exchange.DownloadBLOB(ctx, d.Blob.PayloadRef)
		if err != nil {
			return nil, i18n.WrapError(ctx, err, i18n.MsgDownloadBlobFailed, d.Blob.PayloadRef)
		}
		defer reader.Close()

		// ... to the public storage
		publicRef, err := bm.publicstorage.PublishData(ctx, reader)
		if err != nil {
			return nil, err
		}
		log.L(ctx).Infof("Published blob with hash '%s' for data '%s' to public storage: '%s'", d.Data.Blob, d.Data.ID, publicRef)

		// Update the data in the database, with the public reference.
		// We do this independently for each piece of data
		update := database.DataQueryFactory.NewUpdate(ctx).Set("blob.public", publicRef)
		err = bm.database.UpdateData(ctx, d.Data.ID, update)
		if err != nil {
			return nil, err
		}

	}

	// Now we broadcast the message, as all data has been published
	if err := bm.broadcastMessageCommon(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
