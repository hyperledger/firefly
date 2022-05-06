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

package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	urlGetNamespaces     = "/namespaces"
	urlUploadData        = "/namespaces/default/data"
	urlGetMessages       = "/namespaces/default/messages"
	urlBroadcastMessage  = "/namespaces/default/messages/broadcast"
	urlPrivateMessage    = "/namespaces/default/messages/private"
	urlRequestMessage    = "/namespaces/default/messages/requestreply"
	urlGetData           = "/namespaces/default/data"
	urlGetDataBlob       = "/namespaces/default/data/%s/blob"
	urlGetEvents         = "/namespaces/default/events"
	urlSubscriptions     = "/namespaces/default/subscriptions"
	urlDatatypes         = "/namespaces/default/datatypes"
	urlIdentities        = "/namespaces/default/identities"
	urlIdentity          = "/namespaces/default/identities/%s"
	urlVerifiers         = "/namespaces/default/verifiers"
	urlTokenPools        = "/namespaces/default/tokens/pools"
	urlTokenMint         = "/namespaces/default/tokens/mint"
	urlTokenBurn         = "/namespaces/default/tokens/burn"
	urlTokenTransfers    = "/namespaces/default/tokens/transfers"
	urlTokenApprovals    = "/namespaces/default/tokens/approvals"
	urlTokenAccounts     = "/namespaces/default/tokens/accounts"
	urlTokenBalances     = "/namespaces/default/tokens/balances"
	urlContractInvoke    = "/namespaces/default/contracts/invoke"
	urlContractQuery     = "/namespaces/default/contracts/query"
	urlContractInterface = "/namespaces/default/contracts/interfaces"
	urlContractListeners = "/namespaces/default/contracts/listeners"
	urlContractAPI       = "/namespaces/default/apis"
	urlBlockchainEvents  = "/namespaces/default/blockchainevents"
	urlGetOrganizations  = "/namespaces/default/network/organizations"
	urlGetOrgKeys        = "/namespaces/default/identities/%s/verifiers"
)

func NewResty(t *testing.T) *resty.Client {
	client := resty.New()
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		if os.Getenv("NAMESPACE") != "" {
			req.URL = strings.Replace(req.URL, "/namespaces/default", "/namespaces/"+os.Getenv("NAMESPACE"), 1)
		}
		t.Logf("==> %s %s %s", req.Method, req.URL, req.QueryParam)
		return nil
	})
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		if resp == nil {
			return nil
		}
		t.Logf("<== %d", resp.StatusCode())
		if resp.IsError() {
			t.Logf("<!! %s", resp.String())
			t.Logf("Headers: %+v", resp.Header())
		}
		return nil
	})

	return client
}

func GetNamespaces(client *resty.Client) (*resty.Response, error) {
	return client.R().
		SetResult(&[]core.Namespace{}).
		Get(urlGetNamespaces)
}

func CreateNamespaces(client *resty.Client, namespace string) (*resty.Response, error) {
	namespaceJSON := map[string]interface{}{
		"description": "Firefly test namespace",
		"name":        namespace,
	}
	return client.R().
		SetBody(namespaceJSON).
		Post(urlGetNamespaces)
}

func GetMessageEvents(t *testing.T, client *resty.Client, startTime time.Time, topic string, expectedStatus int) (events []*core.EnrichedEvent) {
	path := urlGetEvents
	resp, err := client.R().
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetQueryParam("topic", topic).
		SetQueryParam("sort", "sequence").
		SetQueryParam("fetchreferences", "true").
		SetResult(&events).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s (count=%d)", path, resp.StatusCode(), resp.String(), len(events))
	return events
}

func GetMessages(t *testing.T, client *resty.Client, startTime time.Time, msgType core.MessageType, topic string, expectedStatus int) (msgs []*core.Message) {
	path := urlGetMessages
	resp, err := client.R().
		SetQueryParam("type", string(msgType)).
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetQueryParam("topics", topic).
		SetQueryParam("sort", "confirmed").
		SetResult(&msgs).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return msgs
}

func GetData(t *testing.T, client *resty.Client, startTime time.Time, expectedStatus int) (data core.DataArray) {
	path := urlGetData
	resp, err := client.R().
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetResult(&data).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return data
}

func GetDataForMessage(t *testing.T, client *resty.Client, startTime time.Time, msgID *fftypes.UUID) (data core.DataArray) {
	path := urlGetMessages
	path += "/" + msgID.String() + "/data"
	resp, err := client.R().
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetResult(&data).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return data
}

func GetBlob(t *testing.T, client *resty.Client, data *core.Data, expectedStatus int) []byte {
	path := fmt.Sprintf(urlGetDataBlob, data.ID)
	resp, err := client.R().
		SetDoNotParseResponse(true).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	blob, err := ioutil.ReadAll(resp.RawBody())
	require.NoError(t, err)
	return blob
}

func GetOrgs(t *testing.T, client *resty.Client, expectedStatus int) (orgs []*core.Identity) {
	path := urlGetOrganizations
	resp, err := client.R().
		SetQueryParam("sort", "created").
		SetResult(&orgs).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return orgs
}

func GetIdentityBlockchainKeys(t *testing.T, client *resty.Client, identityID *fftypes.UUID, expectedStatus int) (verifiers []*core.Verifier) {
	path := fmt.Sprintf(urlGetOrgKeys, identityID)
	resp, err := client.R().
		SetQueryParam("type", fmt.Sprintf("!=%s", core.VerifierTypeFFDXPeerID)).
		SetResult(&verifiers).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return verifiers
}

func CreateSubscription(t *testing.T, client *resty.Client, input interface{}, expectedStatus int) *core.Subscription {
	path := urlSubscriptions
	var sub core.Subscription
	resp, err := client.R().
		SetBody(input).
		SetResult(&sub).
		SetHeader("Content-Type", "application/json").
		Post(path)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &sub
}

func CleanupExistingSubscription(t *testing.T, client *resty.Client, namespace, name string) {
	var subs []*core.Subscription
	path := urlSubscriptions
	resp, err := client.R().
		SetResult(&subs).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	for _, s := range subs {
		if s.Namespace == namespace && s.Name == name {
			DeleteSubscription(t, client, s.ID)
		}
	}
}

func DeleteSubscription(t *testing.T, client *resty.Client, id *fftypes.UUID) {
	path := fmt.Sprintf("%s/%s", urlSubscriptions, id)
	resp, err := client.R().Delete(path)
	require.NoError(t, err)
	require.Equal(t, 204, resp.StatusCode(), "DELETE %s [%d]: %s", path, resp.StatusCode(), resp.String())
}

func BroadcastMessage(t *testing.T, client *resty.Client, topic string, data *core.DataRefOrValue, confirm bool) (*resty.Response, error) {
	return BroadcastMessageAsIdentity(t, client, "", topic, data, confirm)
}

func BroadcastMessageAsIdentity(t *testing.T, client *resty.Client, did, topic string, data *core.DataRefOrValue, confirm bool) (*resty.Response, error) {
	var msg core.Message
	res, err := client.R().
		SetBody(core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					Topics: core.FFStringArray{topic},
					SignerRef: core.SignerRef{
						Author: did,
					},
				},
			},
			InlineData: core.InlineData{data},
		}).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&msg).
		Post(urlBroadcastMessage)
	t.Logf("Sent broadcast msg: %s", msg.Header.ID)
	return res, err
}

func ClaimCustomIdentity(t *testing.T, client *resty.Client, key, name, desc string, profile fftypes.JSONObject, parent *fftypes.UUID, confirm bool) *core.Identity {
	var identity core.Identity
	res, err := client.R().
		SetBody(core.IdentityCreateDTO{
			Name:   name,
			Type:   core.IdentityTypeCustom,
			Parent: parent.String(),
			Key:    key,
			IdentityProfile: core.IdentityProfile{
				Description: desc,
				Profile:     profile,
			},
		}).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&identity).
		Post(urlIdentities)
	assert.NoError(t, err)
	assert.True(t, res.IsSuccess())
	t.Logf("Identity creation initiated with key %s: %s (%s)", key, identity.ID, identity.DID)
	return &identity
}

func GetIdentity(t *testing.T, client *resty.Client, id *fftypes.UUID) *core.Identity {
	var identity core.Identity
	res, err := client.R().
		SetResult(&identity).
		Get(fmt.Sprintf(urlIdentity, id))
	assert.NoError(t, err)
	assert.True(t, res.IsSuccess())
	return &identity
}

func GetVerifiers(t *testing.T, client *resty.Client) []*core.Verifier {
	var verifiers []*core.Verifier
	res, err := client.R().
		SetResult(&verifiers).
		Get(urlVerifiers)
	assert.NoError(t, err)
	assert.True(t, res.IsSuccess())
	return verifiers
}

func CreateBlob(t *testing.T, client *resty.Client, dt *core.DatatypeRef) *core.Data {
	r, _ := rand.Int(rand.Reader, big.NewInt(1024*1024))
	blob := make([]byte, r.Int64()+1024*1024)
	for i := 0; i < len(blob); i++ {
		blob[i] = byte('a' + i%26)
	}
	var blobHash fftypes.Bytes32 = sha256.Sum256(blob)
	t.Logf("Blob size=%d hash=%s", len(blob), &blobHash)
	var data core.Data
	formData := map[string]string{}
	if dt == nil {
		// If there's no datatype, tell FireFly to automatically add a data payload
		formData["autometa"] = "true"
		formData["metadata"] = `{"mymeta": "data"}`
	} else {
		// Otherwise use a tagging only approach, where we allow a nil value, specify that this should
		// not be validated, but still set a datatype for classification of the data.
		formData["validator"] = "none"
		formData["datatype.name"] = dt.Name
		formData["datatype.version"] = dt.Version
	}
	resp, err := client.R().
		SetFormData(formData).
		SetFileReader("file", "myfile.txt", bytes.NewReader(blob)).
		SetResult(&data).
		Post(urlUploadData)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode(), "POST %s [%d]: %s", urlUploadData, resp.StatusCode(), resp.String())
	t.Logf("Data created: %s", data.ID)
	if dt == nil {
		assert.Equal(t, "data", data.Value.JSONObject().GetString("mymeta"))
		assert.Equal(t, "myfile.txt", data.Value.JSONObject().GetString("filename"))
		assert.Equal(t, "myfile.txt", data.Blob.Name)
	} else {
		assert.Equal(t, core.ValidatorTypeNone, data.Validator)
		assert.Equal(t, *dt, *data.Datatype)
	}
	assert.Equal(t, int64(len(blob)), data.Blob.Size)
	assert.Equal(t, blobHash, *data.Blob.Hash)
	return &data
}

func BroadcastBlobMessage(t *testing.T, client *resty.Client, topic string) (*core.Data, *resty.Response, error) {
	data := CreateBlob(t, client, nil)
	res, err := client.R().
		SetBody(core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					Topics: core.FFStringArray{topic},
				},
			},
			InlineData: core.InlineData{
				{DataRef: core.DataRef{ID: data.ID}},
			},
		}).
		Post(urlBroadcastMessage)
	return data, res, err
}

func PrivateBlobMessageDatatypeTagged(ts *testState, client *resty.Client, topic string, orgNames []string) (*core.Data, *resty.Response, error) {
	data := CreateBlob(ts.t, client, &core.DatatypeRef{Name: "myblob"})
	members := make([]core.MemberInput, len(orgNames))
	for i, oName := range orgNames {
		// We let FireFly resolve the friendly name of the org to the identity
		members[i] = core.MemberInput{
			Identity: oName,
		}
	}
	res, err := client.R().
		SetBody(core.MessageInOut{
			Message: core.Message{
				Header: core.MessageHeader{
					Topics: core.FFStringArray{topic},
				},
			},
			InlineData: core.InlineData{
				{DataRef: core.DataRef{ID: data.ID}},
			},
			Group: &core.InputGroup{
				Members: members,
				Name:    fmt.Sprintf("test_%d", ts.startTime.UnixNano()),
			},
		}).
		Post(urlPrivateMessage)
	return data, res, err
}

func PrivateMessage(ts *testState, client *resty.Client, topic string, data *core.DataRefOrValue, orgNames []string, tag string, txType core.TransactionType, confirm bool) (*resty.Response, error) {
	return PrivateMessageWithKey(ts, client, "", topic, data, orgNames, tag, txType, confirm)
}

func PrivateMessageWithKey(ts *testState, client *resty.Client, key, topic string, data *core.DataRefOrValue, orgNames []string, tag string, txType core.TransactionType, confirm bool) (*resty.Response, error) {
	members := make([]core.MemberInput, len(orgNames))
	for i, oName := range orgNames {
		// We let FireFly resolve the friendly name of the org to the identity
		members[i] = core.MemberInput{
			Identity: oName,
		}
	}
	msg := core.MessageInOut{
		Message: core.Message{
			Header: core.MessageHeader{
				Tag:    tag,
				TxType: txType,
				Topics: core.FFStringArray{topic},
				SignerRef: core.SignerRef{
					Key: key,
				},
			},
		},
		InlineData: core.InlineData{data},
		Group: &core.InputGroup{
			Members: members,
			Name:    fmt.Sprintf("test_%d", ts.startTime.UnixNano()),
		},
	}
	res, err := client.R().
		SetBody(msg).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&msg.Message).
		Post(urlPrivateMessage)
	ts.t.Logf("Sent private message %s to %+v", msg.Header.ID, msg.Group.Members)
	return res, err
}

func RequestReply(ts *testState, client *resty.Client, data *core.DataRefOrValue, orgNames []string, tag string, txType core.TransactionType) *core.MessageInOut {
	members := make([]core.MemberInput, len(orgNames))
	for i, oName := range orgNames {
		// We let FireFly resolve the friendly name of the org to the identity
		members[i] = core.MemberInput{
			Identity: oName,
		}
	}
	msg := core.MessageInOut{
		Message: core.Message{
			Header: core.MessageHeader{
				Tag:    tag,
				TxType: txType,
			},
		},
		InlineData: core.InlineData{data},
		Group: &core.InputGroup{
			Members: members,
			Name:    fmt.Sprintf("test_%d", ts.startTime.UnixNano()),
		},
	}
	var replyMsg core.MessageInOut
	resp, err := client.R().
		SetBody(msg).
		SetResult(&replyMsg).
		Post(urlRequestMessage)
	require.NoError(ts.t, err)
	require.Equal(ts.t, 200, resp.StatusCode(), "POST %s [%d]: %s", urlUploadData, resp.StatusCode(), resp.String())
	return &replyMsg
}

func CreateDatatype(t *testing.T, client *resty.Client, datatype *core.Datatype, confirm bool) *core.Datatype {
	var dtReturn core.Datatype
	path := urlDatatypes
	resp, err := client.R().
		SetBody(datatype).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&dtReturn).
		Post(path)
	require.NoError(t, err)
	expected := 202
	if confirm {
		expected = 200
	}
	require.Equal(t, expected, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &dtReturn
}

func CreateTokenPool(t *testing.T, client *resty.Client, pool *core.TokenPool, confirm bool) *core.TokenPool {
	var poolOut core.TokenPool
	path := urlTokenPools
	resp, err := client.R().
		SetBody(pool).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&poolOut).
		Post(path)
	require.NoError(t, err)
	expected := 202
	if confirm {
		expected = 200
	}
	require.Equal(t, expected, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &poolOut
}

func GetTokenPools(t *testing.T, client *resty.Client, startTime time.Time) (pools []*core.TokenPool) {
	path := urlTokenPools
	resp, err := client.R().
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetResult(&pools).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return pools
}

func MintTokens(t *testing.T, client *resty.Client, mint *core.TokenTransferInput, confirm bool) *core.TokenTransfer {
	var transferOut core.TokenTransfer
	path := urlTokenMint
	resp, err := client.R().
		SetBody(mint).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&transferOut).
		Post(path)
	require.NoError(t, err)
	expected := 202
	if confirm {
		expected = 200
	}
	require.Equal(t, expected, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &transferOut
}

func BurnTokens(t *testing.T, client *resty.Client, burn *core.TokenTransferInput, confirm bool) *core.TokenTransfer {
	var transferOut core.TokenTransfer
	path := urlTokenBurn
	resp, err := client.R().
		SetBody(burn).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&transferOut).
		Post(path)
	require.NoError(t, err)
	expected := 202
	if confirm {
		expected = 200
	}
	require.Equal(t, expected, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &transferOut
}

func TransferTokens(t *testing.T, client *resty.Client, transfer *core.TokenTransferInput, confirm bool) *core.TokenTransfer {
	var transferOut core.TokenTransfer
	path := urlTokenTransfers
	resp, err := client.R().
		SetBody(transfer).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&transferOut).
		Post(path)
	require.NoError(t, err)
	expected := 202
	if confirm {
		expected = 200
	}
	require.Equal(t, expected, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &transferOut
}

func GetTokenTransfers(t *testing.T, client *resty.Client, poolID *fftypes.UUID) (transfers []*core.TokenTransfer) {
	path := urlTokenTransfers
	resp, err := client.R().
		SetQueryParam("pool", poolID.String()).
		SetResult(&transfers).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return transfers
}

func TokenApproval(t *testing.T, client *resty.Client, approval *core.TokenApprovalInput, confirm bool) *core.TokenApproval {
	var approvalOut core.TokenApproval
	path := urlTokenApprovals
	resp, err := client.R().
		SetBody(approval).
		SetQueryParam("confirm", strconv.FormatBool(confirm)).
		SetResult(&approvalOut).
		Post(path)
	require.NoError(t, err)
	expected := 202
	if confirm {
		expected = 200
	}
	require.Equal(t, expected, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &approvalOut
}

func GetTokenApprovals(t *testing.T, client *resty.Client, poolID *fftypes.UUID) (approvals []*core.TokenApproval) {
	path := urlTokenApprovals
	resp, err := client.R().
		SetQueryParam("pool", poolID.String()).
		SetQueryParam("active", "true").
		SetResult(&approvals).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return approvals
}

func GetTokenAccounts(t *testing.T, client *resty.Client, poolID *fftypes.UUID) (accounts []*core.TokenAccount) {
	path := urlTokenAccounts
	resp, err := client.R().
		SetResult(&accounts).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return accounts
}

func GetTokenAccountPools(t *testing.T, client *resty.Client, identity string) (pools []*core.TokenAccountPool) {
	path := urlTokenAccounts + "/" + identity + "/pools"
	resp, err := client.R().
		SetQueryParam("sort", "-updated").
		SetResult(&pools).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return pools
}

func GetTokenBalance(t *testing.T, client *resty.Client, poolID *fftypes.UUID, tokenIndex, key string) (account *core.TokenBalance) {
	var accounts []*core.TokenBalance
	path := urlTokenBalances
	resp, err := client.R().
		SetQueryParam("pool", poolID.String()).
		SetQueryParam("tokenIndex", tokenIndex).
		SetQueryParam("key", key).
		SetResult(&accounts).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	require.Equal(t, len(accounts), 1)
	return accounts[0]
}

func CreateContractListener(t *testing.T, client *resty.Client, event *core.FFIEvent, location *fftypes.JSONObject) *core.ContractListener {
	body := core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Location: fftypes.JSONAnyPtr(location.String()),
			Event: &core.FFISerializedEvent{
				FFIEventDefinition: event.FFIEventDefinition,
			},
			Topic: "firefly-e2e",
		},
	}
	var sub core.ContractListener
	path := urlContractListeners
	resp, err := client.R().
		SetBody(&body).
		SetResult(&sub).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &sub
}

func CreateFFIContractListener(t *testing.T, client *resty.Client, ffiReference *core.FFIReference, eventPath string, location *fftypes.JSONObject) *core.ContractListener {
	body := core.ContractListenerInput{
		ContractListener: core.ContractListener{
			Location:  fftypes.JSONAnyPtr(location.String()),
			Interface: ffiReference,
			Topic:     "firefly-e2e",
		},
		EventPath: eventPath,
	}
	var listener core.ContractListener
	path := urlContractListeners
	resp, err := client.R().
		SetBody(&body).
		SetResult(&listener).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &listener
}

func GetContractListeners(t *testing.T, client *resty.Client, startTime time.Time) (subs []*core.ContractListener) {
	path := urlContractListeners
	resp, err := client.R().
		SetQueryParam("created", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetResult(&subs).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return subs
}

func GetContractEvents(t *testing.T, client *resty.Client, startTime time.Time, subscriptionID *fftypes.UUID) (events []*core.BlockchainEvent) {
	path := urlBlockchainEvents
	resp, err := client.R().
		SetQueryParam("timestamp", fmt.Sprintf(">%d", startTime.UnixNano())).
		SetQueryParam("subscriptionId", subscriptionID.String()).
		SetResult(&events).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return events
}

func DeleteContractListener(t *testing.T, client *resty.Client, id *fftypes.UUID) {
	path := urlContractListeners + "/" + id.String()
	resp, err := client.R().Delete(path)
	require.NoError(t, err)
	require.Equal(t, 204, resp.StatusCode(), "DELETE %s [%d]: %s", path, resp.StatusCode(), resp.String())
}

func InvokeContractMethod(t *testing.T, client *resty.Client, req *core.ContractCallRequest) (interface{}, error) {
	var res interface{}
	path := urlContractInvoke
	resp, err := client.R().
		SetBody(req).
		SetResult(&res).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 202, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func QueryContractMethod(t *testing.T, client *resty.Client, req *core.ContractCallRequest) (interface{}, error) {
	var res interface{}
	path := urlContractQuery
	resp, err := client.R().
		SetBody(req).
		SetResult(&res).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func CreateFFI(t *testing.T, client *resty.Client, ffi *core.FFI) (interface{}, error) {
	var res interface{}
	path := urlContractInterface
	resp, err := client.R().
		SetBody(ffi).
		SetResult(&res).
		SetQueryParam("confirm", "true").
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func CreateContractAPI(t *testing.T, client *resty.Client, name string, FFIReference *core.FFIReference, location *fftypes.JSONAny) (interface{}, error) {
	apiReqBody := &core.ContractAPI{
		Name:      name,
		Interface: FFIReference,
		Location:  location,
	}

	var res interface{}
	path := urlContractAPI
	resp, err := client.R().
		SetBody(apiReqBody).
		SetResult(&res).
		SetQueryParam("confirm", "true").
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func InvokeContractAPIMethod(t *testing.T, client *resty.Client, APIName string, methodName string, input *fftypes.JSONAny) (interface{}, error) {
	apiReqBody := map[string]interface{}{
		"input": input,
	}
	var res interface{}
	path := fmt.Sprintf("%s/%s/invoke/%s", urlContractAPI, APIName, methodName)
	resp, err := client.R().
		SetBody(apiReqBody).
		SetResult(&res).
		SetQueryParam("confirm", "true").
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func QueryContractAPIMethod(t *testing.T, client *resty.Client, APIName string, methodName string, input *fftypes.JSONAny) (interface{}, error) {
	apiReqBody := map[string]interface{}{
		"input": input,
	}
	var res interface{}
	path := fmt.Sprintf("%s/%s/query/%s", urlContractAPI, APIName, methodName)
	resp, err := client.R().
		SetBody(apiReqBody).
		SetResult(&res).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func CreateContractAPIListener(t *testing.T, client *resty.Client, APIName, eventName, topic string) (*core.ContractListener, error) {
	apiReqBody := map[string]interface{}{
		"topic": topic,
	}
	var listener core.ContractListener
	path := fmt.Sprintf("%s/%s/listeners/%s", urlContractAPI, APIName, eventName)
	resp, err := client.R().
		SetBody(apiReqBody).
		SetResult(&listener).
		Post(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "POST %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return &listener, err
}

func GetEvent(t *testing.T, client *resty.Client, eventID string) (interface{}, error) {
	var res interface{}
	path := fmt.Sprintf("%s/%s", urlGetEvents, eventID)
	resp, err := client.R().
		SetResult(&res).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}

func GetBlockchainEvent(t *testing.T, client *resty.Client, eventID string) (interface{}, error) {
	var res interface{}
	path := fmt.Sprintf("%s/%s", urlBlockchainEvents, eventID)
	resp, err := client.R().
		SetResult(&res).
		Get(path)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode(), "GET %s [%d]: %s", path, resp.StatusCode(), resp.String())
	return res, err
}
