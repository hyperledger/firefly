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

package networkmap

import (
	"context"
	"fmt"

	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

type DIDDocument struct {
	Context             []string              `json:"@context"`
	ID                  string                `json:"id"`
	Authentication      []string              `json:"authentication"`
	VerificationMethods []*VerificationMethod `json:"verificationMethod"`
}

type VerificationMethod struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Controller string `json:"controller"`
	// Controler specific fields
	BlockchainAccountID string `json:"blockchainAcountId,omitempty"`
	MSPIdentityString   string `json:"mspIdentityString,omitempty"`
	DataExchangePeerID  string `json:"dataExchangePeerID,omitempty"`
}

func (nm *networkMap) generateDIDDocument(ctx context.Context, identity *fftypes.Identity) (doc *DIDDocument, err error) {

	fb := database.VerifierQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("identity", identity.ID),
		fb.Eq("namespace", identity.Namespace),
	)
	verifiers, _, err := nm.database.GetVerifiers(ctx, filter)
	if err != nil {
		return nil, err
	}

	doc = &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		},
		ID: identity.DID,
	}
	doc.VerificationMethods = make([]*VerificationMethod, 0, len(verifiers))
	doc.Authentication = make([]string, 0, len(verifiers))
	for _, verifier := range verifiers {
		vm := nm.generateDIDAuthentication(ctx, identity, verifier)
		if vm != nil {
			doc.VerificationMethods = append(doc.VerificationMethods, vm)
			doc.Authentication = append(doc.Authentication, fmt.Sprintf("#%s", verifier.Hash.String()))
		}
	}
	return doc, nil
}

func (nm *networkMap) generateDIDAuthentication(ctx context.Context, identity *fftypes.Identity, verifier *fftypes.Verifier) *VerificationMethod {
	switch verifier.Type {
	case fftypes.VerifierTypeEthAddress:
		return nm.generateEthAddressVerifier(identity, verifier)
	case fftypes.VerifierTypeMSPIdentity:
		return nm.generateMSPVerifier(identity, verifier)
	case fftypes.VerifierTypeFFDXPeerID:
		return nm.generateDXPeerIDVerifier(identity, verifier)
	default:
		log.L(ctx).Warnf("Unknown verifier type '%s' on verifier '%s' of DID '%s' (%s) - cannot add to DID document", verifier.Type, verifier.Value, identity.DID, identity.ID)
		return nil
	}
}

func (nm *networkMap) generateEthAddressVerifier(identity *fftypes.Identity, verifier *fftypes.Verifier) *VerificationMethod {
	return &VerificationMethod{
		ID:                  verifier.Hash.String(),
		Type:                "EcdsaSecp256k1VerificationKey2019",
		Controller:          identity.DID,
		BlockchainAccountID: verifier.Value,
	}
}

func (nm *networkMap) generateMSPVerifier(identity *fftypes.Identity, verifier *fftypes.Verifier) *VerificationMethod {
	return &VerificationMethod{
		ID:                verifier.Hash.String(),
		Type:              "HyperledgerFabricMSPIdentity",
		Controller:        identity.DID,
		MSPIdentityString: verifier.Value,
	}
}

func (nm *networkMap) generateDXPeerIDVerifier(identity *fftypes.Identity, verifier *fftypes.Verifier) *VerificationMethod {
	return &VerificationMethod{
		ID:                 verifier.Hash.String(),
		Type:               "FireFlyDataExchangePeerIdentity",
		Controller:         identity.DID,
		DataExchangePeerID: verifier.Value,
	}
}
