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

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/hyperledger/firefly/pkg/database"
)

func (dh *definitionHandlers) handleIdentityClaimBroadcast(ctx context.Context, state *core.BatchState, msg *core.Message, data core.DataArray, verificationID *fftypes.UUID) (HandlerResult, error) {
	var claim core.IdentityClaim
	if valid := dh.getSystemBroadcastPayload(ctx, msg, data, &claim); !valid {
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedBadPayload, "identity claim", msg.Header.ID)
	}
	return dh.handleIdentityClaim(ctx, state, msg, &claim, verificationID)
}

func (dh *definitionHandlers) verifyClaimSignature(ctx context.Context, msg *core.Message, identity *core.Identity, parent *core.Identity) error {
	author := msg.Header.Author
	if author == "" {
		return i18n.NewError(ctx, coremsgs.MsgDefRejectedAuthorBlank, "identity claim", msg.Header.ID)
	}

	var expectedSigner *core.Identity
	switch {
	case identity.Type == core.IdentityTypeNode:
		// In the special case of a node, the parent signs it directly
		expectedSigner = parent
	default:
		expectedSigner = identity
	}

	valid := author == expectedSigner.DID ||
		(expectedSigner.Type == core.IdentityTypeOrg && author == fmt.Sprintf("%s%s", core.FireFlyOrgDIDPrefix, expectedSigner.ID))
	if !valid {
		log.L(ctx).Errorf("unable to process identity claim %s - signature mismatch type=%s author=%s expected=%s", msg.Header.ID, identity.Type, author, expectedSigner.DID)
		return i18n.NewError(ctx, coremsgs.MsgDefRejectedSignatureMismatch, "identity claim", msg.Header.ID)
	}
	return nil
}

func (dh *definitionHandlers) getClaimVerifier(msg *core.Message, identity *core.Identity) *core.Verifier {
	verifier := &core.Verifier{
		Identity:  identity.ID,
		Namespace: identity.Namespace,
	}
	switch identity.Type {
	case core.IdentityTypeNode:
		verifier.VerifierRef.Type = core.VerifierTypeFFDXPeerID
		verifier.VerifierRef.Value = identity.Profile.GetString("id")
	default:
		verifier.VerifierRef.Type = dh.blockchain.VerifierType()
		verifier.VerifierRef.Value = msg.Header.Key
	}
	verifier.Seal()
	return verifier
}

func (dh *definitionHandlers) confirmVerificationForClaim(ctx context.Context, state *core.BatchState, msg *core.Message, identity, parent *core.Identity) (*fftypes.UUID, error) {
	// Query for messages on the topic for this DID, signed by the right identity
	idTopic := identity.Topic()
	fb := database.MessageQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("topics", idTopic),
		fb.Eq("author", parent.DID),
		fb.Eq("type", core.MessageTypeDefinition),
		fb.Eq("state", core.MessageStateConfirmed),
		fb.Eq("tag", core.SystemTagIdentityVerification),
	)
	candidates, _, err := dh.database.GetMessages(ctx, dh.namespace, filter)
	if err != nil {
		return nil, err
	}
	// We also need to check pending messages in the current pin batch
	for _, pending := range state.PendingConfirms {
		if pending.Header.Topics.String() == idTopic &&
			pending.Header.Author == parent.DID &&
			pending.Header.Type == core.MessageTypeDefinition &&
			pending.Header.Tag == core.SystemTagIdentityVerification {
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
			var verification core.IdentityVerification
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

func (dh *definitionHandlers) handleIdentityClaim(ctx context.Context, state *core.BatchState, msg *core.Message, identityClaim *core.IdentityClaim, verificationID *fftypes.UUID) (HandlerResult, error) {
	l := log.L(ctx)

	identity := identityClaim.Identity
	parent, retryable, err := dh.identity.VerifyIdentityChain(ctx, identity)
	if err != nil && retryable {
		return HandlerResult{Action: ActionRetry}, err
	} else if err != nil {
		// This cannot be processed as something in the identity chain is invalid.
		// We treat this as a park - because we don't know if the parent identity
		// will be processed after this message and generate a rewind.
		// They are on separate topics, so there is not ordering assurance between the two messages.
		l.Infof("Unable to process identity claim (parked) %s: %s", msg.Header.ID, err)
		return HandlerResult{Action: ActionWait}, nil
	}

	// For multi-party namespaces, check that the claim message was appropriately signed
	if dh.multiparty {
		if err := dh.verifyClaimSignature(ctx, msg, identity, parent); err != nil {
			return HandlerResult{Action: ActionReject}, err
		}
	}

	existingIdentity, err := dh.database.GetIdentityByName(ctx, identity.Type, identity.Namespace, identity.Name)
	if err == nil && existingIdentity == nil {
		existingIdentity, err = dh.database.GetIdentityByID(ctx, dh.namespace, identity.ID)
	}
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // retry database errors
	}
	if existingIdentity != nil && !existingIdentity.IdentityBase.Equals(ctx, &identity.IdentityBase) {
		// If the existing one matches - this is just idempotent replay. No action needed, just confirm
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedConflict, "identity claim", identity.ID, existingIdentity.ID)
	}

	// Check uniqueness of verifier
	verifier := dh.getClaimVerifier(msg, identity)
	existingVerifier, err := dh.database.GetVerifierByValue(ctx, verifier.Type, identity.Namespace, verifier.Value)
	if err != nil {
		return HandlerResult{Action: ActionRetry}, err // retry database errors
	}
	if existingVerifier != nil && !existingVerifier.Identity.Equals(identity.ID) {
		verifierLabel := fmt.Sprintf("%s:%s", verifier.Type, verifier.Value)
		existingVerifierLabel := fmt.Sprintf("%s:%s", verifier.Type, verifier.Value)
		return HandlerResult{Action: ActionReject}, i18n.NewError(ctx, coremsgs.MsgDefRejectedConflict, "identity verifier", verifierLabel, existingVerifierLabel)
	}

	// For child identities in multi-party namespaces, check that the parent signed a verification message
	if dh.multiparty && parent != nil && identity.Type != core.IdentityTypeNode {
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
			log.L(ctx).Infof("Identity %s (%s) awaiting verification claim='%s'", identity.DID, identity.ID, msg.Header.ID)
			return HandlerResult{Action: ActionConfirm}, nil
		}
		log.L(ctx).Infof("Identity %s (%s) verified claim='%s' verification='%s'", identity.DID, identity.ID, msg.Header.ID, verificationID)
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
	if identity.Type == core.IdentityTypeNode {
		state.AddPreFinalize(
			func(ctx context.Context) error {
				// Tell the data exchange about this node. Treat these errors like database errors - and return for retry processing
				return dh.exchange.AddPeer(ctx, identity.Profile)
			})
	}

	state.AddConfirmedDIDClaim(identity.DID)
	state.AddFinalize(func(ctx context.Context) error {
		event := core.NewEvent(core.EventTypeIdentityConfirmed, identity.Namespace, identity.ID, nil, core.SystemTopicDefinitions)
		return dh.database.InsertEvent(ctx, event)
	})
	return HandlerResult{Action: ActionConfirm}, nil
}
