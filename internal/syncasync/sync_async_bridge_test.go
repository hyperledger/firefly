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

package syncasync

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestSyncAsyncBridge(t *testing.T) (*syncAsyncBridge, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mse := &sysmessagingmocks.SystemEvents{}
	sa := NewSyncAsyncBridge(ctx, mdi, mdm)
	sa.Init(mse)
	return sa.(*syncAsyncBridge), cancel
}

func TestRequestReplyOk(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	var requestID *fftypes.UUID
	replyID := fftypes.NewUUID()
	dataID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetMessageByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		assert.NotNil(t, requestID)
		gmid.ReturnArguments = mock.Arguments{
			&fftypes.Message{
				Header: fftypes.MessageHeader{
					ID:  replyID,
					CID: requestID,
				},
				Data: fftypes.DataRefs{
					{ID: dataID},
				},
			}, nil,
		}
	}

	mdm := sa.data.(*datamocks.Manager)
	mdm.On("GetMessageData", sa.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.Byteable(`"response data"`)},
	}, true, nil)

	reply, err := sa.RequestReply(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		requestID = id
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeMessageConfirmed,
					Reference: replyID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, *replyID, *reply.Header.ID)
	assert.Equal(t, `"response data"`, string(reply.InlineData[0].Value))

}

func TestAwaitConfirmationOk(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	var requestID *fftypes.UUID
	dataID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetMessageByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		assert.NotNil(t, requestID)
		msgSent := &fftypes.Message{}
		msgSent.Header.ID = requestID
		msgSent.Confirmed = fftypes.Now()
		msgSent.Rejected = false
		gmid.ReturnArguments = mock.Arguments{
			msgSent, nil,
		}
	}

	mdm := sa.data.(*datamocks.Manager)
	mdm.On("GetMessageData", sa.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.Byteable(`"response data"`)},
	}, true, nil)

	reply, err := sa.SendConfirm(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		requestID = id
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeMessageConfirmed,
					Reference: requestID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, *requestID, *reply.Header.ID)

}

func TestAwaitConfirmationRejected(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	var requestID *fftypes.UUID
	dataID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetMessageByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		assert.NotNil(t, requestID)
		msgSent := &fftypes.Message{}
		msgSent.Header.ID = requestID
		msgSent.Confirmed = fftypes.Now()
		msgSent.Rejected = false
		gmid.ReturnArguments = mock.Arguments{
			msgSent, nil,
		}
	}

	mdm := sa.data.(*datamocks.Manager)
	mdm.On("GetMessageData", sa.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.Byteable(`"response data"`)},
	}, true, nil)

	_, err := sa.SendConfirm(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		requestID = id
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeMessageRejected,
					Reference: requestID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.Regexp(t, "FF10269", err)
}

func TestRequestReplyTimeout(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	_, err := sa.RequestReply(sa.ctx, "ns1", func(requestID *fftypes.UUID) error {
		return nil
	})
	assert.Regexp(t, "FF10260", err)
}

func TestRequestSetupSystemListenerFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := sa.RequestReply(sa.ctx, "ns1", func(requestID *fftypes.UUID) error {
		return nil
	})
	assert.Regexp(t, "pop", err)

}

func TestEventCallbackNotInflight(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeMessageConfirmed,
		},
	})
	assert.NoError(t, err)

}

func TestEventCallbackWrongType(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeGroupConfirmed,
		},
	})
	assert.NoError(t, err)

}

func TestEventCallbackMsgLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", sa.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeMessageConfirmed,
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestEventCallbackTokenPoolLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", sa.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypePoolConfirmed,
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestEventCallbackTokenTransferLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfer", sa.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeTransferConfirmed,
		},
	})
	assert.EqualError(t, err, "pop")

}

func TestEventCallbackMsgNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", sa.ctx, mock.Anything).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeMessageConfirmed,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestEventCallbackRejectedMsgNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", sa.ctx, mock.Anything).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeMessageRejected,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestEventCallbackTokenPoolNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", sa.ctx, mock.Anything).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypePoolConfirmed,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestEventCallbackTokenTransferNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenTransfer", sa.ctx, mock.Anything).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeTransferConfirmed,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestEventCallbackTokenPoolRejectedNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenPoolByID", sa.ctx, mock.Anything).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypePoolRejected,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestEventCallbackMsgDataLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mdm := sa.data.(*datamocks.Manager)
	mdm.On("GetMessageData", sa.ctx, mock.Anything, true).Return(nil, false, fmt.Errorf("pop"))

	sa.resolveReply(&inflightRequest{}, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:  fftypes.NewUUID(),
			CID: fftypes.NewUUID(),
		},
	})

	mdm.AssertExpectations(t)
}

func TestAwaitTokenPoolConfirmation(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	var requestID *fftypes.UUID

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetTokenPoolByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		assert.NotNil(t, requestID)
		pool := &fftypes.TokenPool{
			ID:   requestID,
			Name: "my-pool",
		}
		gmid.ReturnArguments = mock.Arguments{
			pool, nil,
		}
	}

	reply, err := sa.SendConfirmTokenPool(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		requestID = id
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypePoolConfirmed,
					Reference: requestID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, *requestID, *reply.ID)
	assert.Equal(t, "my-pool", reply.Name)
}

func TestAwaitTokenPoolConfirmationSendFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	_, err := sa.SendConfirmTokenPool(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		return fmt.Errorf("pop")
	})
	assert.EqualError(t, err, "pop")
}

func TestAwaitTokenPoolConfirmationRejected(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	var requestID *fftypes.UUID

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetTokenPoolByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		assert.NotNil(t, requestID)
		pool := &fftypes.TokenPool{
			ID:   requestID,
			Name: "my-pool",
		}
		gmid.ReturnArguments = mock.Arguments{
			pool, nil,
		}
	}

	_, err := sa.SendConfirmTokenPool(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		requestID = id
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypePoolRejected,
					Reference: requestID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.Regexp(t, "FF10276", err)
}

func TestAwaitTokenTransferConfirmation(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	var requestID *fftypes.UUID

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetTokenTransfer", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		assert.NotNil(t, requestID)
		pool := &fftypes.TokenTransfer{
			LocalID:    requestID,
			ProtocolID: "abc",
		}
		gmid.ReturnArguments = mock.Arguments{
			pool, nil,
		}
	}

	reply, err := sa.SendConfirmTokenTransfer(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		requestID = id
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeTransferConfirmed,
					Reference: requestID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, *requestID, *reply.LocalID)
	assert.Equal(t, "abc", reply.ProtocolID)
}

func TestAwaitTokenTransferConfirmationSendFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	_, err := sa.SendConfirmTokenTransfer(sa.ctx, "ns1", func(id *fftypes.UUID) error {
		return fmt.Errorf("pop")
	})
	assert.EqualError(t, err, "pop")
}
