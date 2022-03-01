// Copyright © 2022 Kaleido, Inc.
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
	"encoding/json"
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

	requestID := fftypes.NewUUID()
	replyID := fftypes.NewUUID()
	dataID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetMessageByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
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
		{ID: dataID, Value: fftypes.JSONAnyPtr(`"response data"`)},
	}, true, nil)

	reply, err := sa.WaitForReply(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
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
	assert.Equal(t, `"response data"`, reply.InlineData[0].Value.String())

}

func TestAwaitConfirmationOk(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	dataID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetMessageByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		msgSent := &fftypes.Message{}
		msgSent.Header.ID = requestID
		msgSent.Confirmed = fftypes.Now()
		msgSent.State = fftypes.MessageStateConfirmed
		gmid.ReturnArguments = mock.Arguments{
			msgSent, nil,
		}
	}

	mdm := sa.data.(*datamocks.Manager)
	mdm.On("GetMessageData", sa.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.JSONAnyPtr(`"response data"`)},
	}, true, nil)

	reply, err := sa.WaitForMessage(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
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

	requestID := fftypes.NewUUID()
	dataID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetMessageByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		msgSent := &fftypes.Message{}
		msgSent.Header.ID = requestID
		msgSent.Confirmed = fftypes.Now()
		msgSent.State = fftypes.MessageStateConfirmed
		gmid.ReturnArguments = mock.Arguments{
			msgSent, nil,
		}
	}

	mdm := sa.data.(*datamocks.Manager)
	mdm.On("GetMessageData", sa.ctx, mock.Anything, true).Return([]*fftypes.Data{
		{ID: dataID, Value: fftypes.JSONAnyPtr(`"response data"`)},
	}, true, nil)

	_, err := sa.WaitForMessage(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
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

	_, err := sa.WaitForReply(sa.ctx, "ns1", fftypes.NewUUID(), func(ctx context.Context) error {
		return nil
	})
	assert.Regexp(t, "FF10260", err)
}

func TestRequestSetupSystemListenerFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(fmt.Errorf("pop"))

	_, err := sa.WaitForReply(sa.ctx, "ns1", fftypes.NewUUID(), func(ctx context.Context) error {
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

func TestEventCallbackTokenApprovalLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenApproval", sa.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeApprovalConfirmed,
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

func TestEventCallbackTokenApprovalNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetTokenApproval", sa.ctx, mock.Anything).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeApprovalConfirmed,
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestEventCallbackTokenPoolRejectedNoData(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeDefinition,
			Tag:  string(fftypes.SystemTagDefinePool),
		},
		Data: fftypes.DataRefs{},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", sa.ctx, mock.Anything).Return(msg, nil)

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

func TestEventCallbackTokenPoolRejectedDataError(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	responseID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*responseID: &inflightRequest{},
		},
	}

	dataID := fftypes.NewUUID()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeDefinition,
			Tag:  string(fftypes.SystemTagDefinePool),
		},
		Data: fftypes.DataRefs{
			{ID: dataID},
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", sa.ctx, mock.Anything).Return(msg, nil)
	mdi.On("GetDataByID", sa.ctx, dataID, true).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			Namespace: "ns1",
			ID:        fftypes.NewUUID(),
			Reference: fftypes.NewUUID(),
			Type:      fftypes.EventTypeMessageRejected,
		},
	})
	assert.EqualError(t, err, "pop")

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

	requestID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetTokenPoolByID", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		pool := &fftypes.TokenPool{
			ID:   requestID,
			Name: "my-pool",
		}
		gmid.ReturnArguments = mock.Arguments{
			pool, nil,
		}
	}

	reply, err := sa.WaitForTokenPool(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
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

	_, err := sa.WaitForTokenPool(sa.ctx, "ns1", fftypes.NewUUID(), func(ctx context.Context) error {
		return fmt.Errorf("pop")
	})
	assert.EqualError(t, err, "pop")
}

func TestAwaitTokenPoolConfirmationRejected(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	pool := &fftypes.TokenPoolAnnouncement{
		Pool: &fftypes.TokenPool{
			ID: fftypes.NewUUID(),
		},
	}
	poolJSON, _ := json.Marshal(pool)
	data := &fftypes.Data{
		ID:    fftypes.NewUUID(),
		Value: fftypes.JSONAnyPtrBytes(poolJSON),
	}
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID:   fftypes.NewUUID(),
			Type: fftypes.MessageTypeDefinition,
			Tag:  string(fftypes.SystemTagDefinePool),
		},
		Data: fftypes.DataRefs{
			{ID: data.ID},
		},
	}

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetMessageByID", sa.ctx, msg.Header.ID).Return(msg, nil)
	mdi.On("GetDataByID", sa.ctx, data.ID, true).Return(data, nil)

	_, err := sa.WaitForTokenPool(sa.ctx, "ns1", pool.Pool.ID, func(ctx context.Context) error {
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeMessageRejected,
					Reference: msg.Header.ID,
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

	requestID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetTokenTransfer", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		transfer := &fftypes.TokenTransfer{
			LocalID:    requestID,
			ProtocolID: "abc",
		}
		gmid.ReturnArguments = mock.Arguments{
			transfer, nil,
		}
	}

	reply, err := sa.WaitForTokenTransfer(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
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

func TestAwaitTokenApprovalConfirmation(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	gmid := mdi.On("GetTokenApproval", sa.ctx, mock.Anything)
	gmid.RunFn = func(a mock.Arguments) {
		approval := &fftypes.TokenApproval{
			LocalID:    requestID,
			ProtocolID: "abc",
		}
		gmid.ReturnArguments = mock.Arguments{
			approval, nil,
		}
	}

	reply, err := sa.WaitForTokenApproval(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeApprovalConfirmed,
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

func TestAwaitTokenApprovalConfirmationSendFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	_, err := sa.WaitForTokenApproval(sa.ctx, "ns1", fftypes.NewUUID(), func(ctx context.Context) error {
		return fmt.Errorf("pop")
	})
	assert.EqualError(t, err, "pop")
}

func TestAwaitTokenTransferConfirmationSendFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	_, err := sa.WaitForTokenTransfer(sa.ctx, "ns1", fftypes.NewUUID(), func(ctx context.Context) error {
		return fmt.Errorf("pop")
	})
	assert.EqualError(t, err, "pop")
}

func TestAwaitFailedTokenTransfer(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": requestID.String(),
		},
	}

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(op, nil)

	_, err := sa.WaitForTokenTransfer(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeTransferOpFailed,
					Reference: op.ID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.Regexp(t, "FF10291", err)
}

func TestAwaitFailedTokenApproval(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": requestID.String(),
		},
	}

	mse := sa.sysevents.(*sysmessagingmocks.SystemEvents)
	mse.On("AddSystemEventListener", "ns1", mock.Anything).Return(nil)

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(op, nil)

	_, err := sa.WaitForTokenApproval(sa.ctx, "ns1", requestID, func(ctx context.Context) error {
		go func() {
			sa.eventCallback(&fftypes.EventDelivery{
				Event: fftypes.Event{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.EventTypeApprovalOpFailed,
					Reference: op.ID,
					Namespace: "ns1",
				},
			})
		}()
		return nil
	})
	assert.Regexp(t, "FF10291", err)
}

func TestFailedTokenTransferOpError(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*requestID: &inflightRequest{},
		},
	}

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": requestID.String(),
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.EventTypeTransferOpFailed,
			Reference: op.ID,
			Namespace: "ns1",
		},
	})
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestFailedTokenApprovalOpError(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*requestID: &inflightRequest{},
		},
	}

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": requestID.String(),
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(nil, fmt.Errorf("pop"))

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.EventTypeApprovalOpFailed,
			Reference: op.ID,
			Namespace: "ns1",
		},
	})
	assert.EqualError(t, err, "pop")

	mdi.AssertExpectations(t)
}

func TestFailedTokenApprovalOpNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*requestID: &inflightRequest{},
		},
	}

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": requestID.String(),
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.EventTypeApprovalOpFailed,
			Reference: op.ID,
			Namespace: "ns1",
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestFailedTokenApprovalIDLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*requestID: &inflightRequest{},
		},
	}

	op := &fftypes.Operation{
		ID:    fftypes.NewUUID(),
		Input: fftypes.JSONObject{},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(op, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.EventTypeApprovalOpFailed,
			Reference: op.ID,
			Namespace: "ns1",
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestFailedTokenTransferOpNotFound(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*requestID: &inflightRequest{},
		},
	}

	op := &fftypes.Operation{
		ID: fftypes.NewUUID(),
		Input: fftypes.JSONObject{
			"localId": requestID.String(),
		},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(nil, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.EventTypeTransferOpFailed,
			Reference: op.ID,
			Namespace: "ns1",
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}

func TestFailedTokenTransferIDLookupFail(t *testing.T) {

	sa, cancel := newTestSyncAsyncBridge(t)
	defer cancel()

	requestID := fftypes.NewUUID()
	sa.inflight = map[string]map[fftypes.UUID]*inflightRequest{
		"ns1": {
			*requestID: &inflightRequest{},
		},
	}

	op := &fftypes.Operation{
		ID:    fftypes.NewUUID(),
		Input: fftypes.JSONObject{},
	}

	mdi := sa.database.(*databasemocks.Plugin)
	mdi.On("GetOperationByID", sa.ctx, op.ID).Return(op, nil)

	err := sa.eventCallback(&fftypes.EventDelivery{
		Event: fftypes.Event{
			ID:        fftypes.NewUUID(),
			Type:      fftypes.EventTypeTransferOpFailed,
			Reference: op.ID,
			Namespace: "ns1",
		},
	})
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
}
