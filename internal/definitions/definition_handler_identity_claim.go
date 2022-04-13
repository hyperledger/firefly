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
	"fmt"

	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/log"
)

func (dh *definitionHandlers) handleIdentityClaimBroadcast(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, data fftypes.DataArray, verificationID *fftypes.UUID) (HandlerResult, error) {
	var claim fftypes.IdentityClaim
	valid := dh.getSystemBroadcastPayload(ctx, msg, data, &claim)
	if !valid {
		return HandlerResult{Action: ActionReject}, nil
	}

	return dh.handleIdentityClaim(ctx, state, msg, &claim, verificationID)

}

func (dh *definitionHandlers) verifyClaimSignature(ctx context.Context, msg *fftypes.Message, identity *fftypes.Identity, parent *fftypes.Identity) (valid bool) {

	author := msg.Header.Author
	if author == "" {
		return false
	}

	var expectedSigner *fftypes.Identity
	switch {
	case identity.Type == fftypes.IdentityTypeNode:
		// In the special case of a node, the parent signs it directly
		expectedSigner = parent
	default:
		expectedSigner = identity
	}

	valid = author == expectedSigner.DID ||
		(expectedSigner.Type == fftypes.IdentityTypeOrg && author == fmt.Sprintf("%s%s", fftypes.FireFlyOrgDIDPrefix, expectedSigner.ID))
	if !valid {
		log.L(ctx).Warnf("Unable to process identity claim %s - signature mismatch type=%s author=%s expected=%s", msg.Header.ID, identity.Type, author, expectedSigner.DID)
	}
	return valid
}

func (dh *definitionHandlers) getClaimVerifier(msg *fftypes.Message, identity *fftypes.Identity) *fftypes.Verifier {
	verifier := &fftypes.Verifier{
		Identity:  identity.ID,
		Namespace: identity.Namespace,
	}
	switch identity.Type {
	case fftypes.IdentityTypeNode:
		verifier.VerifierRef.Type = fftypes.VerifierTypeFFDXPeerID
		verifier.VerifierRef.Value = identity.Profile.GetString("id")
	default:
		verifier.VerifierRef.Type = dh.blockchain.VerifierType()
		verifier.VerifierRef.Value = msg.Header.Key
	}
	verifier.Seal()
	return verifier
}

func (dh *definitionHandlers) confirmVerificationForClaim(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, identity, parent *fftypes.Identity) (*fftypes.UUID, error) {
	// Query for messages on the topic for this DID, signed by the right identity
	idTopic := identity.Topic()
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("topics", idTopic),
		fb.Eq("author", parent.DID),
		fb.Eq("type", fftypes.MessageTypeDefinition),
		fb.Eq("state", fftypes.MessageStateConfirmed),
		fb.Eq("tag", fftypes.SystemTagIdentityVerification),
	)
	candidates, _, err := dh.database.GetMessages(ctx, filter)
	if err != nil {
		return nil, err
	}
	// We also need to check pending messages in the current pin batch
	for _, pending := range state.GetPendingConfirm() {
		if pending.Header.Topics.String() == idTopic &&
			pending.Header.Author == parent.DID &&
			pending.Header.Type == fftypes.MessageTypeDefinition &&
			pending.Header.Tag == fftypes.SystemTagIdentityVerification {
			candidates = append(candidates, pending)
		}
	}
	for _, candidate := range candidates {
		data, foundAll, err := dh.data.GetMessageDataCached(ctx, candidate)
		if err != nil {
			return nil, err
		}
		identityMatches := false
		var verificationID *fftypes.UUID
		var verificationHash *fftypes.Bytes32
		if foundAll {
			var verification fftypes.IdentityVerification
			if !dh.getSystemBroadcastPayload(ctx, msg, data, &verification) {
				return nil, nil
			}
			identityMatches = verification.Identity.Equals(ctx, &identity.IdentityBase)
			verificationID = verification.Claim.ID
			verificationHash = verification.Claim.Hash
			if identityMatches && msg.Header.ID.Equals(verificationID) && msg.Hash.Equals(verificationHash) {
				return candidate.Header.ID, nil
			}
		}
		log.L(ctx).Warnf("Skipping invalid potential verification '%s' for identity claimID='%s' claimHash=%s: foundData=%t identityMatch=%t id=%s hash=%s", candidate.Header.ID, msg.Header.ID, msg.Hash, foundAll, identityMatches, verificationID, verificationHash)
	}
	return nil, nil
}

func (dh *definitionHandlers) handleIdentityClaim(ctx context.Context, state DefinitionBatchState, msg *fftypes.Message, identityClaim *fftypes.IdentityClaim, verificationID *fftypes.UUID) (HandlerResult, error) {
	l := log.L(ctx)

	identity := identityClaim.Identity
	parent, _, retryable, err := dh.identity.VerifyIdentityChain(ctx, identity)
	if err != nil && retryable {
		return HandlerResult{Action: ActionRetry}, err
	} else if err != nil {
		// This cannot be processed as the parent does not exist (or similar).
		// We treat this as a bad request, as nodes should not be broadcast until the parent identity is
		// is already confirmed. (Note different processing for org/custom child's, where there's a parent
		// verification to coordinate).
		l.Warnf("Unable to process identity claim %s: %s", msg.Header.ID, err)
		return HandlerResult{Action: ActionReject}, nil
	}

	// Check signature verification
	if !dh.verifyClaimSignature(ctx, msg, identity, parent) {
		return HandlerResult{Action: ActionReject}, nil
	}

	existingIdentity, err := dh.database.GetIdentityByName(ctx, identity.Type, identity.Namespace, identity.Name)
	if err == nil && existingIdentity == nil {
		existingIdentity, err = dh.database.GetIdentityByID(ctx, identity.ID)
	}
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // retry database errors
	}
	if existingIdentity != nil && !existingIdentity.IdentityBase.Equals(ctx, &identity.IdentityBase) {
		// If the existing one matches - this is just idempotent replay. No action needed, just confirm
		l.Warnf("Unable to process identity claim %s - conflict with existing: %v", msg.Header.ID, existingIdentity.ID)
		return HandlerResult{Action: ActionReject}, nil
	}

	// Check uniqueness of verifier
	verifier := dh.getClaimVerifier(msg, identity)
	existingVerifier, err := dh.database.GetVerifierByValue(ctx, verifier.Type, identity.Namespace, verifier.Value)
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // retry database errors
	}
	if existingVerifier != nil && !existingVerifier.Identity.Equals(identity.ID) {
		log.L(ctx).Warnf("Unable to process identity claim %s - verifier type=%s value=%s already registered: %v", msg.Header.ID, verifier.Type, verifier.Value, existingVerifier.Hash)
		return HandlerResult{Action: ActionReject}, nil
	}

	if parent != nil && identity.Type != fftypes.IdentityTypeNode {
		// The verification might be passed into this function, if we confirm the verification second,
		// or we might have to hunt for it, if we confirm the verification first.
		if verificationID == nil {
			// Search for a corresponding verification message on the same topic
			verificationID, err = dh.confirmVerificationForClaim(ctx, state, msg, identity, parent)
			if err != nil {
				return HandlerResult{Action: ActionRetry}, err // retry database errors
			}
		}
		if verificationID == nil {
			// Ok, we still confirm the message as it's valid, and we do not want to block the context.
			// But we do NOT go on to create the identity - we will be called back
			return HandlerResult{Action: ActionConfirm}, nil
		}
		log.L(ctx).Infof("Identity '%s' verified claim='%s' verification='%s'", identity.ID, msg.Header.ID, verificationID)
		identity.Messages.Verification = verificationID
	}

	if existingVerifier == nil {
		if err = dh.database.UpsertVerifier(ctx, verifier, database.UpsertOptimizationNew); err != nil {
			return HandlerResult{Action: ActionRetry}, err
		}
	}
	if existingIdentity == nil {
		if err = dh.database.UpsertIdentity(ctx, identity, database.UpsertOptimizationNew); err != nil {
			return HandlerResult{Action: ActionRetry}, err
		}
	}

	// If this is a node, we need to add that peer
	if identity.Type == fftypes.IdentityTypeNode {
		state.AddPreFinalize(
			func(ctx context.Context) error {
				// Tell the data exchange about this node. Treat these errors like database errors - and return for retry processing
				return dh.exchange.AddPeer(ctx, identity.Profile)
			})
	}

	state.AddFinalize(func(ctx context.Context) error {
		event := fftypes.NewEvent(fftypes.EventTypeIdentityConfirmed, identity.Namespace, identity.ID, nil, fftypes.SystemTopicDefinitions)
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, nil
}
