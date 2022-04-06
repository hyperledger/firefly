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

package events

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/mock"
)

func TestBlobReceiverBackgroundDispatchOK(t *testing.T) {

	em, cancel := newTestEventManagerWithDBConcurrency(t)
	defer cancel()
	em.blobReceiver.start()

	dataID := fftypes.NewUUID()
	batchID := fftypes.NewUUID()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobs", mock.Anything, mock.Anything).Return([]*fftypes.Blob{}, nil, nil)
	mdi.On("InsertBlobs", mock.Anything, mock.Anything).Return(nil, nil)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).Return(fftypes.DataRefs{
		{ID: dataID},
	}, nil, nil)
	mdi.On("GetMessagesForData", mock.Anything, dataID, mock.Anything).Return([]*fftypes.Message{
		{Header: fftypes.MessageHeader{ID: fftypes.NewUUID()}, BatchID: batchID},
	}, nil, nil)

	blobHash := fftypes.NewRandB32()
	done := make(chan struct{})
	em.blobReceiver.blobReceived(em.ctx, &blobNotification{
		blob: &fftypes.Blob{
			Hash: blobHash,
		},
	})
	em.blobReceiver.blobReceived(em.ctx, &blobNotification{
		blob: &fftypes.Blob{
			Hash: blobHash, // de-dup'd
		},
		onComplete: func() {
			close(done)
		},
	})
	<-done

	mdi.AssertExpectations(t)
	em.blobReceiver.stop()

}

func TestBlobReceiverBackgroundDispatchCancelled(t *testing.T) {

	em, cancel := newTestEventManagerWithDBConcurrency(t)
	cancel()
	em.blobReceiver.start()

	em.blobReceiver.blobReceived(em.ctx, &blobNotification{
		blob: &fftypes.Blob{
			Hash: fftypes.NewRandB32(),
		},
	})
	em.blobReceiver.stop()

}

func TestBlobReceiverBackgroundDispatchFail(t *testing.T) {

	em, cancel := newTestEventManagerWithDBConcurrency(t)
	em.blobReceiver.start()

	done := make(chan struct{})
	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobs", mock.Anything, mock.Anything).Return(nil, nil, fmt.Errorf("pop")).Run(func(args mock.Arguments) {
		cancel()
		close(done)
	})

	em.blobReceiver.blobReceived(em.ctx, &blobNotification{
		blob: &fftypes.Blob{
			Hash: fftypes.NewRandB32(),
		},
	})
	<-done

	mdi.AssertExpectations(t)
	em.blobReceiver.stop()

}

func TestBlobReceiverDispatchDup(t *testing.T) {

	em, cancel := newTestEventManager(t)
	defer cancel()

	blobHash := fftypes.NewRandB32()

	mdi := em.database.(*databasemocks.Plugin)
	mdi.On("GetBlobs", mock.Anything, mock.Anything).Return([]*fftypes.Blob{
		{Hash: blobHash, PayloadRef: "payload1"},
	}, nil, nil)

	em.blobReceiver.blobReceived(em.ctx, &blobNotification{
		blob: &fftypes.Blob{
			Hash:       blobHash,
			PayloadRef: "payload1",
		},
	})

	mdi.AssertExpectations(t)

}
