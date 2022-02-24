// Copyright © 2022 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in comdiliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or imdilied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assets

import (
	"context"
	"fmt"
	"testing"

	"github.com/hyperledger/firefly/internal/syncasync"
	"github.com/hyperledger/firefly/mocks/databasemocks"
	"github.com/hyperledger/firefly/mocks/datamocks"
	"github.com/hyperledger/firefly/mocks/identitymanagermocks"
	"github.com/hyperledger/firefly/mocks/syncasyncmocks"
	"github.com/hyperledger/firefly/mocks/tokenmocks"
	"github.com/hyperledger/firefly/mocks/txcommonmocks"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetTokenApprovals(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	mdi := am.database.(*databasemocks.Plugin)
	fb := database.TokenApprovalQueryFacory.NewFilter(context.Background())
	f := fb.And()
	mdi.On("GetTokenApprovals", context.Background(), f).Return([]*fftypes.TokenApproval{}, nil, nil)
	_, _, err := am.GetTokenApprovals(context.Background(), "ns1", f)
	assert.NoError(t, err)
}

func TestTokenApprovalSuccess(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mti.On("TokensApproval", context.Background(), mock.Anything, "F1", &approval.TokenApproval).Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.NoError(t, err)
}

func TestTokenApprovalSuccessUnknownIdentity(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mti.On("TokensApproval", context.Background(), mock.Anything, "F1", &approval.TokenApproval).Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.NoError(t, err)
}

func TestApprovalUnknownConnectorNoConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}

	am.tokens = make(map[string]tokens.Plugin)

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.Regexp(t, "FF10292", err)
}

func TestApprovalUnknownConnectorMultipleConnectors(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}

	am.tokens["magic-tokens"] = nil
	am.tokens["magic-tokens2"] = nil

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.Regexp(t, "FF10292", err)
}

func TestApprovalUnknownConnectorBadNamespace(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}

	am.tokens = make(map[string]tokens.Plugin)

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)

	_, err := am.TokenApproval(context.Background(), "", approval, false)
	assert.Regexp(t, "FF10131", err)
}

func TestApprovalBadConnector(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved:  true,
			Operator:  "operator",
			Key:       "key",
			Connector: "bad",
		},
		Pool: "pool1",
	}

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.Regexp(t, "FF10272", err)
}

func TestApprovalUnknownPoolSuccess(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And()
	f.Limit(1).Count(true)
	tokenPools := []*fftypes.TokenPool{
		{
			Name:       "pool1",
			ProtocolID: "F1",
			State:      fftypes.TokenPoolStateConfirmed,
		},
	}
	totalCount := int64(1)
	filterResult := &database.FilterResult{
		TotalCount: &totalCount,
	}
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPools", context.Background(), mock.MatchedBy((func(f database.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(tokenPools[0], nil)
	mti.On("TokensApproval", context.Background(), mock.Anything, "F1", &approval.TokenApproval).Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.NoError(t, err)
}

func TestApprovalUnknownPoolNoPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	fb := database.TokenPoolQueryFactory.NewFilter(context.Background())
	f := fb.And()
	f.Limit(1).Count(true)
	tokenPools := []*fftypes.TokenPool{}
	totalCount := int64(0)
	filterResult := &database.FilterResult{
		TotalCount: &totalCount,
	}
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPools", context.Background(), mock.MatchedBy((func(f database.AndFilter) bool {
		info, _ := f.Finalize()
		return info.Count && info.Limit == 1
	}))).Return(tokenPools, filterResult, nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.Regexp(t, "FF10292", err)
}

func TestApprovalBadPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(nil, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.EqualError(t, err, "pop")
}

func TestApprovalUnconfirmedPool(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStatePending,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.Regexp(t, "FF10293", err)

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestApprovalIdentityFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool: "pool1",
	}

	mim := am.identity.(*identitymanagermocks.Manager)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "").Return("", fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.EqualError(t, err, "pop")
}

func TestApprovalFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mti.On("TokensApproval", context.Background(), mock.Anything, "F1", &approval.TokenApproval).Return(fmt.Errorf("pop"))
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mth.On("WriteOperationFailure", context.Background(), mock.Anything, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.EqualError(t, err, "pop")
}

func TestApprovalTransactionFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(nil, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.EqualError(t, err, "pop")

	mim.AssertExpectations(t)
	mdi.AssertExpectations(t)
}

func TestApprovalFailAndDbFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mti.On("TokensApproval", context.Background(), mock.Anything, "F1", &approval.TokenApproval).Return(fmt.Errorf("pop"))
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mth.On("WriteOperationFailure", context.Background(), mock.Anything, fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.EqualError(t, err, "pop")
}

func TestApprovalOperationsFail(t *testing.T) {
	am, cancel := newTestAssets(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mth := am.txHelper.(*txcommonmocks.Helper)

	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(fmt.Errorf("pop"))

	_, err := am.TokenApproval(context.Background(), "ns1", approval, false)
	assert.EqualError(t, err, "pop")
}

func TestTokenApprovalConfirm(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved: true,
			Operator: "operator",
			Key:      "key",
		},
		Pool: "pool1",
	}
	pool := &fftypes.TokenPool{
		ProtocolID: "F1",
		State:      fftypes.TokenPoolStateConfirmed,
	}

	mdi := am.database.(*databasemocks.Plugin)
	mti := am.tokens["magic-tokens"].(*tokenmocks.Plugin)
	mim := am.identity.(*identitymanagermocks.Manager)
	mdm := am.data.(*datamocks.Manager)
	msa := am.syncasync.(*syncasyncmocks.Bridge)
	mth := am.txHelper.(*txcommonmocks.Helper)
	mim.On("ResolveInputSigningKeyOnly", context.Background(), "key").Return("0x12345", nil)
	mdi.On("GetTokenPool", context.Background(), "ns1", "pool1").Return(pool, nil)
	mti.On("TokensApproval", context.Background(), mock.Anything, "F1", &approval.TokenApproval).Return(nil)
	mth.On("SubmitNewTransaction", context.Background(), "ns1", fftypes.TransactionTypeTokenApproval).Return(fftypes.NewUUID(), nil)
	mdi.On("InsertOperation", context.Background(), mock.Anything).Return(nil)

	msa.On("WaitForTokenApproval", context.Background(), "ns1", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			send := args[3].(syncasync.RequestSender)
			send(context.Background())
		}).
		Return(&fftypes.TokenApproval{}, nil)

	_, err := am.TokenApproval(context.Background(), "ns1", approval, true)
	assert.NoError(t, err)

	mdi.AssertExpectations(t)
	mdm.AssertExpectations(t)
	msa.AssertExpectations(t)
	mti.AssertExpectations(t)
}

func TestApprovalPrepare(t *testing.T) {
	am, cancel := newTestAssetsWithMetrics(t)
	defer cancel()

	approval := &fftypes.TokenApprovalInput{
		TokenApproval: fftypes.TokenApproval{
			Approved:  true,
			Operator:  "operator",
			Key:       "key",
			Connector: "magic-tokens",
		},
		Pool: "pool1",
	}

	sender := am.NewApproval("ns1", approval)
	err := sender.Prepare(context.Background())
	assert.NoError(t, err)
}
