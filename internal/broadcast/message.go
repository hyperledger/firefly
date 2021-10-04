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

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (bm *broadcastManager) BroadcastMessage(ctx context.Context, ns string, in *fftypes.MessageInOut, waitConfirm bool) (out *fftypes.Message, err error) {
	return bm.BroadcastMessageWithID(ctx, ns, nil, in, nil, waitConfirm)
}

func (bm *broadcastManager) BroadcastMessageWithID(ctx context.Context, ns string, id *fftypes.UUID, unresolved *fftypes.MessageInOut, resolved *fftypes.Message, waitConfirm bool) (out *fftypes.Message, err error) {
	if unresolved != nil {
		resolved = &unresolved.Message
	}
	resolved.Header.ID = id
	resolved.Header.Namespace = ns
	resolved.Header.Type = fftypes.MessageTypeBroadcast
	if resolved.Header.TxType == "" {
		resolved.Header.TxType = fftypes.TransactionTypeBatchPin
	}

	// Resolve the sending identity
	if err := bm.identity.ResolveInputIdentity(ctx, &resolved.Header.Identity); err != nil {
		return nil, i18n.WrapError(ctx, err, i18n.MsgAuthorInvalid)
	}

	// We optimize the DB storage of all the parts of the message using transaction semantics (assuming those are supported by the DB plugin
	var dataToPublish []*fftypes.DataAndBlob
	err = bm.database.RunAsGroup(ctx, func(ctx context.Context) error {
		if unresolved != nil {
			// The data manager is responsible for the heavy lifting of storing/validating all our in-line data elements
			resolved.Data, dataToPublish, err = bm.data.ResolveInlineDataBroadcast(ctx, ns, unresolved.InlineData)
			if err != nil {
				return err
			}
		}

		// If we have data to publish, we break out of the DB transaction, do the publishes, then
		// do the send later - as that could take a long time (multiple seconds) depending on the size
		//
		// Same for waiting for confirmation of the send from the blockchain
		if len(dataToPublish) > 0 || waitConfirm {
			return nil
		}

		out, err = bm.broadcastMessageCommon(ctx, resolved, false)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Perform deferred processing
	if len(dataToPublish) > 0 {
		return bm.publishBlobsAndSend(ctx, resolved, dataToPublish, waitConfirm)
	} else if waitConfirm {
		return bm.broadcastMessageCommon(ctx, resolved, true)
	}

	// The broadcastMessage function modifies the input message to create all the refs
	return out, err
}

func (bm *broadcastManager) publishBlobsAndSend(ctx context.Context, msg *fftypes.Message, dataToPublish []*fftypes.DataAndBlob, waitConfirm bool) (*fftypes.Message, error) {

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
	return bm.broadcastMessageCommon(ctx, msg, waitConfirm)
}
