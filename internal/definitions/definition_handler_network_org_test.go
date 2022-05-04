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

package definitions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const oldOrgExampleMessage = `{
    "header": {
      "id": "6c138623-d995-4c40-b17b-81a00e7d150c",
      "type": "definition",
      "txtype": "batch_pin",
      "author": "did:firefly:org/f08153cc-c605-4239-9087-e08747e1fb4e",
      "key": "0x214840c7c62cddf7a854a830d55018b38e4e78be",
      "created": "2022-02-24T15:30:58.985588799Z",
      "namespace": "ff_system",
      "topics": [
        "ff_organizations"
      ],
      "tag": "ff_define_organization",
      "datahash": "ab59ea680208bef4a303f9c637efca1766228c8a8ae69881da5cc5f0512e1e1e"
    },
    "hash": "be3ed2c1445c961d897f13f042e002d523d607f0557fd1c6f4597057606129dc",
    "batch": "55cccf24-599e-4295-bb7a-6144cee6d91a",
    "state": "confirmed",
    "confirmed": "2022-02-24T15:31:01.656922592Z",
    "data": [
      {
        "id": "cb87d285-bbcc-4a0e-b9a2-89877894b89a",
        "hash": "ee2241d6dc61fe2288b7abe65daace1e2ae18bfadbf5d905e98f4b25cdf64f9f",
        "validator": "definition",
        "value": {
          "id": "f08153cc-c605-4239-9087-e08747e1fb4e",
          "identity": "0x214840c7c62cddf7a854a830d55018b38e4e78be",
          "name": "org_0",
          "created": "2022-02-24T15:30:58.974970049Z"
        }
      }
    ]
  }`

func testDeprecatedRootOrg(t *testing.T) (*fftypes.DeprecatedOrganization, *fftypes.Message, *fftypes.Data) {

	var msgInOut fftypes.MessageInOut
	err := json.Unmarshal([]byte(oldOrgExampleMessage), &msgInOut)
	assert.NoError(t, err)

	var org fftypes.DeprecatedOrganization
	err = json.Unmarshal(msgInOut.InlineData[0].Value.Bytes(), &org)
	assert.NoError(t, err)

	return &org, &msgInOut.Message, &fftypes.Data{
		ID:        msgInOut.InlineData[0].ID,
		Validator: msgInOut.InlineData[0].Validator,
		Namespace: msgInOut.Header.Namespace,
		Hash:      msgInOut.InlineData[0].Hash,
		Value:     msgInOut.InlineData[0].Value,
	}
}

func TestHandleDeprecatedOrgDefinitionOK(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	org, msg, data := testDeprecatedRootOrg(t)

	mim := dh.identity.(*identitymanagermocks.Manager)
	mim.On("VerifyIdentityChain", ctx, mock.Anything).Return(nil, false, nil)

	mdi := dh.database.(*databasemocks.Plugin)
	mdi.On("GetIdentityByName", ctx, fftypes.IdentityTypeOrg, fftypes.SystemNamespace, org.Name).Return(nil, nil)
	mdi.On("GetIdentityByID", ctx, org.ID).Return(nil, nil)
	mdi.On("GetVerifierByValue", ctx, fftypes.VerifierTypeEthAddress, fftypes.SystemNamespace, msg.Header.Key).Return(nil, nil)
	mdi.On("UpsertIdentity", ctx, mock.MatchedBy(func(identity *fftypes.Identity) bool {
		assert.Equal(t, *msg.Header.ID, *identity.Messages.Claim)
		return true
	}), database.UpsertOptimizationNew).Return(nil)
	mdi.On("UpsertVerifier", ctx, mock.MatchedBy(func(verifier *fftypes.Verifier) bool {
		assert.Equal(t, fftypes.VerifierTypeEthAddress, verifier.Type)
		assert.Equal(t, msg.Header.Key, verifier.Value)
		assert.Equal(t, *org.ID, *verifier.Identity)
		return true
	}), database.UpsertOptimizationNew).Return(nil)
	mdi.On("InsertEvent", mock.Anything, mock.MatchedBy(func(event *fftypes.Event) bool {
		return event.Type == fftypes.EventTypeIdentityConfirmed
	})).Return(nil)

	action, err := dh.HandleDefinitionBroadcast(ctx, bs, msg, fftypes.DataArray{data}, fftypes.NewUUID())
	assert.Equal(t, HandlerResult{Action: ActionConfirm}, action)
	assert.NoError(t, err)

	err = bs.finalizers[0](ctx)
	assert.NoError(t, err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestHandleDeprecatedOrgDefinitionBadData(t *testing.T) {
	dh, bs := newTestDefinitionHandler(t)
	ctx := context.Background()

	action, err := dh.handleDeprecatedOrganizationBroadcast(ctx, bs, &fftypes.Message{}, fftypes.DataArray{})
	assert.Equal(t, HandlerResult{Action: ActionReject}, action)
	assert.NoError(t, err)

	bs.assertNoFinalizers()
}
