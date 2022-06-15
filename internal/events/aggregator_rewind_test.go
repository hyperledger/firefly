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
	"time"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/data"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRewinderE2E(t *testing.T) {
	ag, cancel := newTestAggregator()
	ag.rewinder.minRewindTimeout = 1 * time.Millisecond
	dataID := fftypes.NewUUID()
	batchID1 := fftypes.NewUUID()
	batchID2 := fftypes.NewUUID()
	batchID3 := fftypes.NewUUID()
	batchID4 := fftypes.NewUUID()
	batchID5 := fftypes.NewUUID()

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).
		Return(core.DataRefs{{ID: dataID}}, nil, nil)
	mdi.On("GetBatchIDsForDataAttachments", mock.Anything, []*fftypes.UUID{dataID}).
		Return([]*fftypes.UUID{batchID2}, nil)
	mdm.On("PeekMessageCache", mock.Anything, mock.Anything, data.CRORequireBatchID).Return(nil, nil)
	mdi.On("GetBatchIDsForMessages", mock.Anything, mock.Anything).
		Return([]*fftypes.UUID{batchID3}, nil).Once()
	mdi.On("GetMessageIDs", mock.Anything, "ns1", mock.Anything).
		Return([]*core.IDAndSequence{{ID: *fftypes.NewUUID()}}, nil).Once()
	mdi.On("GetBatchIDsForMessages", mock.Anything, mock.Anything).
		Return([]*fftypes.UUID{batchID4}, nil).Once()

	ag.rewinder.start()

	ag.queueBatchRewind(batchID1)
	ag.queueBlobRewind(fftypes.NewRandB32())
	ag.queueMessageRewind(fftypes.NewUUID())
	ag.queueDIDRewind("did:firefly:org/bob")
	ag.queueBatchRewind(batchID5)

	allReceived := make(map[fftypes.UUID]bool)
	for len(allReceived) < 4 {
		time.Sleep(1 * time.Millisecond)
		batchIDs := ag.rewinder.popRewinds()
		for _, bid := range batchIDs {
			allReceived[bid.(fftypes.UUID)] = true
		}
	}
	assert.True(t, allReceived[*batchID1])
	assert.True(t, allReceived[*batchID2])
	assert.True(t, allReceived[*batchID3])
	assert.True(t, allReceived[*batchID4])
	assert.True(t, allReceived[*batchID5])

	cancel()
	<-ag.rewinder.loop1Done
	<-ag.rewinder.loop2Done

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestProcessStagedRewindsErrorMessages(t *testing.T) {

	ag, cancel := newTestAggregator()
	cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	mockRunAsGroupPassthrough(mdi)
	mdm.On("PeekMessageCache", mock.Anything, mock.Anything, data.CRORequireBatchID).Return(nil, nil)
	mdi.On("GetBatchIDsForMessages", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("pop"))

	ag.rewinder.stagedRewinds = []*rewind{
		{rewindType: rewindMessage},
	}
	ag.rewinder.processStagedRewinds()

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestProcessStagedRewindsMessagesCached(t *testing.T) {

	ag, cancel := newTestAggregator()
	cancel()

	mdi := ag.database.(*databasemocks.Plugin)
	mdm := ag.data.(*datamocks.Manager)

	mockRunAsGroupPassthrough(mdi)
	mdm.On("PeekMessageCache", mock.Anything, mock.Anything, data.CRORequireBatchID).Return(&core.Message{
		BatchID: fftypes.NewUUID(),
	}, nil)

	ag.rewinder.stagedRewinds = []*rewind{
		{rewindType: rewindMessage},
	}
	ag.rewinder.processStagedRewinds()

	mdm.AssertExpectations(t)
	mdi.AssertExpectations(t)

}

func TestProcessStagedRewindsErrorBlobBatchIDs(t *testing.T) {

	ag, cancel := newTestAggregator()
	cancel()

	mdi := ag.database.(*databasemocks.Plugin)

	dataID := fftypes.NewUUID()

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).
		Return(core.DataRefs{{ID: dataID}}, nil, nil)
	mdi.On("GetBatchIDsForDataAttachments", mock.Anything, []*fftypes.UUID{dataID}).
		Return(nil, fmt.Errorf("pop"))

	ag.rewinder.stagedRewinds = []*rewind{
		{rewindType: rewindBlob},
	}
	ag.rewinder.processStagedRewinds()

	mdi.AssertExpectations(t)

}

func TestProcessStagedRewindsErrorBlobDataRefs(t *testing.T) {

	ag, cancel := newTestAggregator()
	cancel()

	mdi := ag.database.(*databasemocks.Plugin)

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetDataRefs", mock.Anything, mock.Anything).
		Return(nil, nil, fmt.Errorf("pop"))

	ag.rewinder.stagedRewinds = []*rewind{
		{rewindType: rewindBlob},
	}
	ag.rewinder.processStagedRewinds()

	mdi.AssertExpectations(t)

}

func TestProcessStagedRewindsErrorDIDs(t *testing.T) {

	ag, cancel := newTestAggregator()
	cancel()

	mdi := ag.database.(*databasemocks.Plugin)

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetMessageIDs", mock.Anything, "ns1", mock.Anything).
		Return(nil, fmt.Errorf("pop"))

	ag.rewinder.stagedRewinds = []*rewind{
		{rewindType: rewindDIDConfirmed},
	}
	ag.rewinder.processStagedRewinds()

	mdi.AssertExpectations(t)

}

func TestProcessStagedRewindsNoDIDs(t *testing.T) {

	ag, cancel := newTestAggregator()
	cancel()

	mdi := ag.database.(*databasemocks.Plugin)

	mockRunAsGroupPassthrough(mdi)
	mdi.On("GetMessageIDs", mock.Anything, "ns1", mock.Anything).
		Return([]*core.IDAndSequence{}, nil)

	ag.rewinder.stagedRewinds = []*rewind{
		{rewindType: rewindDIDConfirmed},
	}
	ag.rewinder.processStagedRewinds()

	mdi.AssertExpectations(t)

}

func TestPopRewindsDoublePopNoBlock(t *testing.T) {

	ag, cancel := newTestAggregator()
	defer cancel()

	ag.rewinder.queuedRewinds = []*rewind{{rewindType: rewindBatch}}
	batchIDs := ag.rewinder.popRewinds()
	assert.Empty(t, batchIDs)

	ag.rewinder.queuedRewinds = []*rewind{{rewindType: rewindBatch}}
	batchIDs = ag.rewinder.popRewinds()
	assert.Empty(t, batchIDs)

}
