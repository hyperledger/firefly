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

package batching

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kaleido-io/firefly/internal/fftypes"
	"github.com/kaleido-io/firefly/internal/i18n"
	"github.com/kaleido-io/firefly/internal/log"
	"github.com/kaleido-io/firefly/internal/persistence"
)

func NewBatchManager(ctx context.Context, persistence persistence.Plugin) (BatchManager, error) {
	if persistence == nil {
		return nil, i18n.NewError(ctx, i18n.MsgInitializationNilDepError)
	}
	bm := &batchManager{
		ctx:         ctx,
		persistence: persistence,
		dispatchers: make(map[fftypes.BatchType]*dispatcher),
	}
	return bm, nil
}

type BatchManager interface {
	RegisterDispatcher(batchType fftypes.BatchType, handler DispatchHandler, batchOptions BatchOptions)
	DispatchMessage(ctx context.Context, batchType fftypes.BatchType, msg *fftypes.MessageRefsOnly) (*uuid.UUID, error)
	Close()
}

type batchManager struct {
	ctx         context.Context
	persistence persistence.Plugin
	dispatchers map[fftypes.BatchType]*dispatcher
}

type DispatchHandler func(context.Context, *fftypes.Batch) error

type BatchOptions struct {
	BatchMaxSize   uint
	BatchTimeout   time.Duration
	DisposeTimeout time.Duration
}

type dispatcher struct {
	handler      DispatchHandler
	mux          sync.Mutex
	processors   map[string]*batchProcessor
	batchOptions BatchOptions
}

func (bm *batchManager) RegisterDispatcher(batchType fftypes.BatchType, handler DispatchHandler, batchOptions BatchOptions) {
	bm.dispatchers[batchType] = &dispatcher{
		handler:      handler,
		batchOptions: batchOptions,
		processors:   make(map[string]*batchProcessor),
	}
}

func (bm *batchManager) removeProcessor(dispatcher *dispatcher, key string) {
	dispatcher.mux.Lock()
	delete(dispatcher.processors, key)
	dispatcher.mux.Unlock()
}

func (bm *batchManager) getProcessor(batchType fftypes.BatchType, namespace, author string) (*batchProcessor, error) {
	dispatcher, ok := bm.dispatchers[batchType]
	if !ok {
		return nil, i18n.NewError(bm.ctx, i18n.MsgUnregisteredBatchType, batchType)
	}
	dispatcher.mux.Lock()
	key := fmt.Sprintf("%s/%s", namespace, author)
	processor, ok := dispatcher.processors[key]
	if !ok {
		processor = newBatchProcessor(
			bm.ctx, // Background context, not the call context
			&batchProcessorConf{
				BatchOptions:       dispatcher.batchOptions,
				namespace:          namespace,
				author:             author,
				persitence:         bm.persistence,
				dispatch:           dispatcher.handler,
				processorQuiescing: func() { bm.removeProcessor(dispatcher, key) },
			},
		)
		dispatcher.processors[key] = processor
	}
	dispatcher.mux.Unlock()
	return processor, nil
}

func (bm *batchManager) Close() {
	if bm != nil {
		for _, d := range bm.dispatchers {
			d.mux.Lock()
			for _, p := range d.processors {
				p.close()
			}
			d.mux.Unlock()
		}
	}
	bm = nil
}

func (bm *batchManager) DispatchMessage(ctx context.Context, batchType fftypes.BatchType, msg *fftypes.MessageRefsOnly) (*uuid.UUID, error) {
	l := log.L(ctx)
	processor, err := bm.getProcessor(batchType, msg.Header.Namespace, msg.Header.Author)
	if err != nil {
		return nil, err
	}
	l.Debugf("Dispatching message %s to %s batch", msg.Header.ID, batchType)
	work := &batchWork{
		msg:        msg,
		dispatched: make(chan *uuid.UUID),
	}
	processor.newWork <- work
	select {
	case <-ctx.Done():
		l.Debugf("Dispatch timeout for mesasge %s", msg.Header.ID)
		// Try to avoid the processor picking this up, if it hasn't already (as we've given up waiting)
		work.abandoned = true
		return nil, i18n.NewError(ctx, i18n.MsgBatchDispatchTimeout)
	case batchID := <-work.dispatched:
		l.Debugf("Dispatched mesasge %s to batch %s", msg.Header.ID, batchID)
		return batchID, nil
	}
}
