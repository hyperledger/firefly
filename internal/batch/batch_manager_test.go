// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/txcommon"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/sysmessagingmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestE2EDispatchBroadcast(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	mni.On("GetNodeUUID", mock.Anything).Return(fftypes.NewUUID())
	readyForDispatch := make(chan bool)
	waitForDispatch := make(chan *DispatchState)
	handler := func(ctx context.Context, state *DispatchState) error {
		_, ok := <-readyForDispatch
		if !ok {
			return nil
		}
		assert.Len(t, state.Pins, 2)
		h := sha256.New()
		nonceBytes, _ := hex.DecodeString(
			"746f70696331",
		/*|  topic1   | */
		) // little endian 12345 in 8 byte hex
		h.Write(nonceBytes)
		assert.Equal(t, hex.EncodeToString(h.Sum([]byte{})), state.Pins[0].String())

		h = sha256.New()
		nonceBytes, _ = hex.DecodeString(
			"746f70696332",
		/*|   topic2  | */
		) // little endian 12345 in 8 byte hex
		h.Write(nonceBytes)
		assert.Equal(t, hex.EncodeToString(h.Sum([]byte{})), state.Pins[1].String())

		waitForDispatch <- state
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	bmi, _ := NewBatchManager(ctx, mni, mdi, mdm, txHelper)
	bm := bmi.(*batchManager)

	bm.RegisterDispatcher("utdispatcher", fftypes.TransactionTypeBatchPin, []fftypes.MessageType{fftypes.MessageTypeBroadcast}, handler, DispatcherOptions{
		BatchMaxSize:   2,
		BatchTimeout:   0,
		DisposeTimeout: 10 * time.Millisecond,
	})

	dataID1 := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			TxType:    fftypes.TransactionTypeBatchPin,
			Type:      fftypes.MessageTypeBroadcast,
			ID:        fftypes.NewUUID(),
			Topics:    []string{"topic1", "topic2"},
			Namespace: "ns1",
			SignerRef: fftypes.SignerRef{Author: "did:firefly:org/abcd", Key: "0x12345"},
		},
		Data: fftypes.DataRefs{
			{ID: dataID1, Hash: dataHash},
		},
	}
	data := &fftypes.Data{
		ID:   dataID1,
		Hash: dataHash,
	}
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(fftypes.DataArray{data}, true, nil)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil, nil).Once()
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil) // pins
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		ctx := a.Get(0).(context.Context)
		fn := a.Get(1).(func(context.Context) error)
		fn(ctx)
	}
	mdi.On("UpdateMessages", mock.Anything, mock.MatchedBy(func(f database.Filter) bool {
		fi, err := f.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("( id IN ['%s'] ) && ( state == 'ready' )", msg.Header.ID.String()), fi.String())
		return true
	}), mock.Anything).Return(nil)
	mdi.On("InsertTransaction", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil) // transaction submit

	err := bm.Start()
	assert.NoError(t, err)

	bm.NewMessages() <- msg.Sequence

	readyForDispatch <- true

	// Check the status while we know there's a flush going on
	status := bm.Status()
	assert.NotNil(t, status.Processors[0].Status.Flushing)

	b := <-waitForDispatch
	assert.Equal(t, *msg.Header.ID, *b.Payload.Messages[0].Header.ID)
	assert.Equal(t, *data.ID, *b.Payload.Data[0].ID)

	close(readyForDispatch)

	// Wait for the reaping
	for len(bm.getProcessors()) > 0 {
		time.Sleep(1 * time.Millisecond)
		bm.NewMessages() <- msg.Sequence
	}

	cancel()
	bm.WaitStop()

}

func TestE2EDispatchPrivateUnpinned(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	mni.On("GetNodeUUID", mock.Anything).Return(fftypes.NewUUID())
	readyForDispatch := make(chan bool)
	waitForDispatch := make(chan *DispatchState)
	var groupID fftypes.Bytes32
	_ = groupID.UnmarshalText([]byte("44dc0861e69d9bab17dd5e90a8898c2ea156ad04e5fabf83119cc010486e6c1b"))
	handler := func(ctx context.Context, state *DispatchState) error {
		_, ok := <-readyForDispatch
		if !ok {
			return nil
		}
		assert.Len(t, state.Pins, 2)
		h := sha256.New()
		nonceBytes, _ := hex.DecodeString(
			"746f70696331" + "44dc0861e69d9bab17dd5e90a8898c2ea156ad04e5fabf83119cc010486e6c1b" + "6469643a66697265666c793a6f72672f61626364" + "0000000000003039",
		/*|  topic1   |    | ---- group id -------------------------------------------------|   |author'"did:firefly:org/abcd'            |  |i64 nonce (12345) */
		/*|               context                                                           |   |          sender + nonce             */
		) // little endian 12345 in 8 byte hex
		h.Write(nonceBytes)
		assert.Equal(t, hex.EncodeToString(h.Sum([]byte{})), state.Pins[0].String())

		h = sha256.New()
		nonceBytes, _ = hex.DecodeString(
			"746f70696332" + "44dc0861e69d9bab17dd5e90a8898c2ea156ad04e5fabf83119cc010486e6c1b" + "6469643a66697265666c793a6f72672f61626364" + "000000000000303a",
		/*|   topic2  |    | ---- group id -------------------------------------------------|   |author'"did:firefly:org/abcd'            |  |i64 nonce (12346) */
		/*|               context                                                           |   |          sender + nonce             */
		) // little endian 12345 in 8 byte hex
		h.Write(nonceBytes)
		assert.Equal(t, hex.EncodeToString(h.Sum([]byte{})), state.Pins[1].String())
		waitForDispatch <- state
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	bmi, _ := NewBatchManager(ctx, mni, mdi, mdm, txHelper)
	bm := bmi.(*batchManager)

	bm.RegisterDispatcher("utdispatcher", fftypes.TransactionTypeBatchPin, []fftypes.MessageType{fftypes.MessageTypePrivate}, handler, DispatcherOptions{
		BatchMaxSize:   2,
		BatchTimeout:   0,
		DisposeTimeout: 120 * time.Second,
	})

	dataID1 := fftypes.NewUUID()
	dataHash := fftypes.NewRandB32()
	msg := &fftypes.Message{
		Header: fftypes.MessageHeader{
			TxType:    fftypes.TransactionTypeBatchPin,
			Type:      fftypes.MessageTypePrivate,
			ID:        fftypes.NewUUID(),
			Topics:    []string{"topic1", "topic2"},
			Namespace: "ns1",
			SignerRef: fftypes.SignerRef{Author: "did:firefly:org/abcd", Key: "0x12345"},
			Group:     &groupID,
		},
		Data: fftypes.DataRefs{
			{ID: dataID1, Hash: dataHash},
		},
	}
	data := &fftypes.Data{
		ID:   dataID1,
		Hash: dataHash,
	}
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(fftypes.DataArray{data}, true, nil)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil, nil).Once()
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	mdi.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil) // pins
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rag.RunFn = func(a mock.Arguments) {
		ctx := a.Get(0).(context.Context)
		fn := a.Get(1).(func(context.Context) error)
		fn(ctx)
	}
	mdi.On("UpdateMessages", mock.Anything, mock.MatchedBy(func(f database.Filter) bool {
		fi, err := f.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("( id IN ['%s'] ) && ( state == 'ready' )", msg.Header.ID.String()), fi.String())
		return true
	}), mock.Anything).Return(nil)
	ugcn := mdi.On("UpsertNonceNext", mock.Anything, mock.Anything).Return(nil)
	nextNonce := int64(12345)
	ugcn.RunFn = func(a mock.Arguments) {
		a[1].(*fftypes.Nonce).Nonce = nextNonce
		nextNonce++
	}
	mdi.On("InsertTransaction", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil) // transaction submit

	err := bm.Start()
	assert.NoError(t, err)

	bm.NewMessages() <- msg.Sequence

	readyForDispatch <- true
	b := <-waitForDispatch
	assert.Equal(t, *msg.Header.ID, *b.Payload.Messages[0].Header.ID)
	assert.Equal(t, *data.ID, *b.Payload.Data[0].ID)

	// Wait until everything closes
	close(readyForDispatch)
	cancel()
	bm.WaitStop()

}

func TestDispatchUnknownType(t *testing.T) {
	log.SetLevel("debug")
	config.Reset()

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	ctx, cancel := context.WithCancel(context.Background())
	bmi, _ := NewBatchManager(ctx, mni, mdi, mdm, txHelper)
	bm := bmi.(*batchManager)

	msg := &fftypes.Message{}
	mdi.On("GetMessages", mock.Anything, mock.Anything).Return([]*fftypes.Message{msg}, nil, nil).Once()
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(fftypes.DataArray{}, true, nil)

	err := bm.Start()
	assert.NoError(t, err)

	cancel()
	bm.WaitStop()

}

func TestInitFailNoPersistence(t *testing.T) {
	_, err := NewBatchManager(context.Background(), nil, nil, nil, nil)
	assert.Error(t, err)
}

func TestGetInvalidBatchTypeMsg(t *testing.T) {

	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	defer bm.Close()
	_, err := bm.(*batchManager).getProcessor(fftypes.BatchTypeBroadcast, "wrong", nil, "ns1", &fftypes.SignerRef{})
	assert.Regexp(t, "FF10126", err)
}

func TestMessageSequencerCancelledContext(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	mdi.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	defer bm.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bm.(*batchManager).ctx = ctx
	bm.(*batchManager).messageSequencer()
	assert.Equal(t, 1, len(mdi.Calls))
}

func TestMessageSequencerMissingMessageData(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	bm.RegisterDispatcher("utdispatcher", fftypes.TransactionTypeNone, []fftypes.MessageType{fftypes.MessageTypeBroadcast},
		func(c context.Context, state *DispatchState) error {
			return nil
		},
		DispatcherOptions{BatchType: fftypes.BatchTypeBroadcast},
	)

	dataID := fftypes.NewUUID()
	mdi.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).
		Return([]*fftypes.Message{
			{
				Header: fftypes.MessageHeader{
					ID:        fftypes.NewUUID(),
					Type:      fftypes.MessageTypeBroadcast,
					Namespace: "ns1",
					TxType:    fftypes.TransactionTypeNone,
				},
				Data: []*fftypes.DataRef{
					{ID: dataID},
				}},
		}, nil, nil).
		Run(func(args mock.Arguments) {
			bm.Close()
		}).
		Once()
	mdi.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{}, nil, nil)
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything, data.CRORequirePublicBlobRefs).Return(fftypes.DataArray{}, false, nil)

	bm.(*batchManager).messageSequencer()

	bm.WaitStop()

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageSequencerUpdateMessagesFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	mni.On("GetNodeUUID", mock.Anything).Return(fftypes.NewUUID())
	ctx, cancelCtx := context.WithCancel(context.Background())
	bm, _ := NewBatchManager(ctx, mni, mdi, mdm, txHelper)
	bm.RegisterDispatcher("utdispatcher", fftypes.TransactionTypeBatchPin, []fftypes.MessageType{fftypes.MessageTypeBroadcast},
		func(c context.Context, state *DispatchState) error {
			return nil
		},
		DispatcherOptions{BatchMaxSize: 1, DisposeTimeout: 0},
	)

	dataID := fftypes.NewUUID()
	mdi.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				TxType:    fftypes.TransactionTypeBatchPin,
				Type:      fftypes.MessageTypeBroadcast,
				Namespace: "ns1",
			},
			Data: []*fftypes.DataRef{
				{ID: dataID},
			}},
	}, nil, nil)
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(fftypes.DataArray{{ID: dataID}}, true, nil)
	mdm.On("UpdateMessageIfCached", mock.Anything, mock.Anything).Return()
	mdi.On("InsertTransaction", mock.Anything, mock.Anything).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil) // transaction submit
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mdi.On("UpdateMessages", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("fizzle"))
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		ctx := a.Get(0).(context.Context)
		fn := a.Get(1).(func(context.Context) error)
		err, ok := fn(ctx).(error)
		if ok && err.Error() == "fizzle" {
			cancelCtx() // so we only go round once
			bm.Close()
		}
		rag.ReturnArguments = mock.Arguments{err}
	}

	bm.(*batchManager).messageSequencer()

	bm.Close()
	bm.WaitStop()

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageSequencerDispatchFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	mni.On("GetNodeUUID", mock.Anything).Return(fftypes.NewUUID())
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	ctx, cancelCtx := context.WithCancel(context.Background())
	bm, _ := NewBatchManager(ctx, mni, mdi, mdm, txHelper)
	bm.RegisterDispatcher("utdispatcher", fftypes.TransactionTypeBatchPin, []fftypes.MessageType{fftypes.MessageTypeBroadcast},
		func(c context.Context, state *DispatchState) error {
			cancelCtx()
			return fmt.Errorf("fizzle")
		}, DispatcherOptions{BatchMaxSize: 1, DisposeTimeout: 0},
	)

	dataID := fftypes.NewUUID()
	mdi.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				TxType:    fftypes.TransactionTypeBatchPin,
				Type:      fftypes.MessageTypeBroadcast,
				Namespace: "ns1",
			},
			Data: []*fftypes.DataRef{
				{ID: dataID},
			}},
	}, nil, nil)
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(fftypes.DataArray{{ID: dataID}}, true, nil)
	mdi.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	bm.(*batchManager).messageSequencer()

	bm.Close()
	bm.WaitStop()

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestMessageSequencerUpdateBatchFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	mni.On("GetNodeUUID", mock.Anything).Return(fftypes.NewUUID())
	ctx, cancelCtx := context.WithCancel(context.Background())
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(ctx, mni, mdi, mdm, txHelper)
	bm.RegisterDispatcher("utdispatcher", fftypes.TransactionTypeBatchPin, []fftypes.MessageType{fftypes.MessageTypeBroadcast},
		func(c context.Context, state *DispatchState) error {
			return nil
		},
		DispatcherOptions{BatchMaxSize: 1, DisposeTimeout: 0},
	)

	dataID := fftypes.NewUUID()
	mdi.On("GetMessages", mock.Anything, mock.Anything, mock.Anything).Return([]*fftypes.Message{
		{
			Header: fftypes.MessageHeader{
				ID:        fftypes.NewUUID(),
				TxType:    fftypes.TransactionTypeBatchPin,
				Type:      fftypes.MessageTypeBroadcast,
				Namespace: "ns1",
			},
			Data: []*fftypes.DataRef{
				{ID: dataID},
			}},
	}, nil, nil)
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(fftypes.DataArray{{ID: dataID}}, true, nil)
	mdi.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("fizzle"))
	rag := mdi.On("RunAsGroup", mock.Anything, mock.Anything, mock.Anything)
	rag.RunFn = func(a mock.Arguments) {
		ctx := a.Get(0).(context.Context)
		fn := a.Get(1).(func(context.Context) error)
		err, ok := fn(ctx).(error)
		if ok && err.Error() == "fizzle" {
			cancelCtx() // so we only go round once
			bm.Close()
		}
		rag.ReturnArguments = mock.Arguments{err}
	}
	mdi.On("InsertTransaction", mock.Anything, mock.Anything).Return(nil).Maybe()
	mdi.On("InsertEvent", mock.Anything, mock.Anything).Return(nil).Maybe()

	bm.(*batchManager).messageSequencer()

	bm.Close()
	bm.WaitStop()

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
}

func TestWaitForPollTimeout(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	bm.(*batchManager).messagePollTimeout = 1 * time.Microsecond
	bm.(*batchManager).waitForNewMessages()
}

func TestWaitForNewMessage(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bmi, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	bm := bmi.(*batchManager)
	bm.readOffset = 22222
	bm.NewMessages() <- 12345
	bm.waitForNewMessages()
	assert.Equal(t, int64(12344), bm.readOffset)
}

func TestAssembleMessageDataNilData(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	bm.Close()
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(nil, false, nil)
	_, err := bm.(*batchManager).assembleMessageData(fftypes.BatchTypePrivate, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{{ID: nil}},
	})
	assert.Regexp(t, "FF10133", err)
}

func TestGetMessageDataFail(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything).Return(nil, false, fmt.Errorf("pop"))
	bm.Close()
	_, _ = bm.(*batchManager).assembleMessageData(fftypes.BatchTypePrivate, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	})
	mdm.AssertExpectations(t)
}

func TestGetMessageNotFound(t *testing.T) {
	mdi := &databasemocks.Plugin{}
	mdm := &datamocks.Manager{}
	mni := &sysmessagingmocks.LocalNodeInfo{}
	txHelper := txcommon.NewTransactionHelper(mdi, mdm)
	bm, _ := NewBatchManager(context.Background(), mni, mdi, mdm, txHelper)
	mdm.On("GetMessageDataCached", mock.Anything, mock.Anything, data.CRORequirePublicBlobRefs).Return(nil, false, nil)
	bm.Close()
	_, err := bm.(*batchManager).assembleMessageData(fftypes.BatchTypeBroadcast, &fftypes.Message{
		Header: fftypes.MessageHeader{
			ID: fftypes.NewUUID(),
		},
		Data: fftypes.DataRefs{
			{ID: fftypes.NewUUID(), Hash: fftypes.NewRandB32()},
		},
	})
	assert.Regexp(t, "FF10133", err)
}
