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

package networkmap

import (
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDIDGenerationOK(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	org1 := testOrg("org1")

	verifierEth := (&fftypes.Verifier{
		Identity:  org1.ID,
		Namespace: org1.Namespace,
		VerifierRef: fftypes.VerifierRef{
			Type:  fftypes.VerifierTypeEthAddress,
			Value: "0xc90d94dE1021fD17fAA2F1FC4F4D36Dff176120d",
		},
		Created: fftypes.Now(),
	}).Seal()
	verifierMSP := (&fftypes.Verifier{
		Identity:  org1.ID,
		Namespace: org1.Namespace,
		VerifierRef: fftypes.VerifierRef{
			Type:  fftypes.VerifierTypeMSPIdentity,
			Value: "mspIdForAcme::x509::CN=fabric-ca::CN=user1",
		},
		Created: fftypes.Now(),
	}).Seal()
	verifierDX := (&fftypes.Verifier{
		Identity:  org1.ID,
		Namespace: org1.Namespace,
		VerifierRef: fftypes.VerifierRef{
			Type:  fftypes.VerifierTypeFFDXPeerID,
			Value: "peer1",
		},
		Created: fftypes.Now(),
	}).Seal()
	verifierUnknown := (&fftypes.Verifier{
		Identity:  org1.ID,
		Namespace: org1.Namespace,
		VerifierRef: fftypes.VerifierRef{
			Type:  fftypes.VerifierType("unknown"),
			Value: "ignore me",
		},
		Created: fftypes.Now(),
	}).Seal()

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", nm.ctx, mock.Anything).Return(org1, nil)
	mdi.On("GetVerifiers", nm.ctx, mock.Anything).Return([]*fftypes.Verifier{
		verifierEth,
		verifierMSP,
		verifierDX,
		verifierUnknown,
	}, nil, nil)

	doc, err := nm.GetDIDDocForIndentityByID(nm.ctx, org1.Namespace, org1.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		},
		ID: org1.DID,
		VerificationMethods: []*VerificationMethod{
			{
				ID:                  verifierEth.Hash.String(),
				Type:                "EcdsaSecp256k1VerificationKey2019",
				Controller:          org1.DID,
				BlockchainAccountID: verifierEth.Value,
			},
			{
				ID:                verifierMSP.Hash.String(),
				Type:              "HyperledgerFabricMSPIdentity",
				Controller:        org1.DID,
				MSPIdentityString: verifierMSP.Value,
			},
			{
				ID:                 verifierDX.Hash.String(),
				Type:               "FireFlyDataExchangePeerIdentity",
				Controller:         org1.DID,
				DataExchangePeerID: verifierDX.Value,
			},
		},
		Authentication: []string{
			fmt.Sprintf("#%s", verifierEth.Hash.String()),
			fmt.Sprintf("#%s", verifierMSP.Hash.String()),
			fmt.Sprintf("#%s", verifierDX.Hash.String()),
		},
	}, doc)

	mdi.AssertExpectations(t)
}

func TestDIDGenerationGetVerifiersFail(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	org1 := testOrg("org1")

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", nm.ctx, mock.Anything).Return(org1, nil)
	mdi.On("GetVerifiers", nm.ctx, mock.Anything).Return(nil, nil, fmt.Errorf("pop"))

	_, err := nm.GetDIDDocForIndentityByID(nm.ctx, org1.Namespace, org1.ID.String())
	assert.Regexp(t, "pop", err)
}

func TestDIDGenerationGetIdentityFail(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()

	org1 := testOrg("org1")

	mdi := nm.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByID", nm.ctx, mock.Anything).Return(nil, fmt.Errorf("pop"))

	_, err := nm.GetDIDDocForIndentityByID(nm.ctx, org1.Namespace, org1.ID.String())
	assert.Regexp(t, "pop", err)
}
