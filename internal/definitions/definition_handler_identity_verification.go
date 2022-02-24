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

package definitions

import (
	"context"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func (dh *definitionHandlers) handleIdentityVerificationBroadcast(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, data []*fftypes.Data) (DefinitionMessageAction, error) {
	var verification fftypes.IdentityVerification
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &verification)
	if !valid {
		return ActionReject, nil
	}

	// See if we find the message to which it refers
	err := verification.Identity.Validate(ctx)
	if err != nil || verification.Identity.Parent == nil || verification.Claim.ID == nil || verification.Claim.Hash == nil {
		log.L(ctx).Warnf("Invalid verification message %s: %v", msg.Header.ID, err)
		return ActionReject, nil
	}

	// Check the verification is signed by the correct org
	parent, err := dh.identity.CachedIdentityLookupByID(ctx, verification.Identity.Parent)
	if err != nil {
		return ActionRetry, err
	}
	if parent == nil {
		log.L(ctx).Warnf("Invalid verification message %s - parent not found: %s", msg.Header.ID, verification.Identity.Parent)
		return ActionReject, nil
	}
	if parent.DID != msg.Header.Author {
		log.L(ctx).Warnf("Invalid verification message %s - parent '%s' does not match signer '%s'", msg.Header.ID, parent.DID, msg.Header.Author)
		return ActionReject, nil
	}

	// At this point, this is a valid verification, but we don't know if the claim has arrived.
	// It might be being processed in the same pin batch as us - so we can't

	// See if the message has already arrived, if so we need to queue a rewind to it
	claimMsg, err := dh.database.GetMessageByID(ctx, verification.Claim.ID)
	if err != nil {
		return ActionRetry, err
	}

	if claimMsg != nil && (claimMsg.State == fftypes.MessageStateConfirmed || state.IsPendingConfirm(claimMsg.Header.ID)) {
		if !claimMsg.Hash.Equals(verification.Claim.Hash) {
			log.L(ctx).Warnf("Invalid verification message %s - hash mismatch claim=%s verification=%s", msg.Header.ID, claimMsg.Hash, verification.Claim.Hash)
			return ActionReject, nil
		}
		data, foundAll, err := dh.data.GetMessageData(ctx, claimMsg, true)
		if err != nil {
			return ActionRetry, nil
		}
		if foundAll {
			// The verification came in after the messsage, so we need to call the idempotent
			// handler of the claim logic again
			return dh.handleIdentityClaimBroadcast(ctx, state, msg, data, true)
		}
	}

	// Just confirm the verification - when the message is processed it will be
	return ActionConfirm, nil

}
