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
	"testing"

	"github.com/hyperledger-labs/firefly/mocks/databasemocks"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func uuidMatches(id1 *fftypes.UUID) interface{} {
	return mock.MatchedBy(func(id2 *fftypes.UUID) bool { return id1.Equals(id2) })
}

func TestGetOrganizationByIDOk(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	id := fftypes.NewUUID()
	nm.database.(*databasemocks.Plugin).On("GetOrganizationByID", nm.ctx, uuidMatches(id)).Return(&fftypes.Organization{ID: id}, nil)
	res, err := nm.GetOrganizationByID(nm.ctx, id.String())
	assert.NoError(t, err)
	assert.Equal(t, *id, *res.ID)
}

func TestGetOrganizationByIDBadUUID(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	_, err := nm.GetOrganizationByID(nm.ctx, "bad")
	assert.Regexp(t, "FF10142", err)
}

func TestGetNodeByIDOk(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	id := fftypes.NewUUID()
	nm.database.(*databasemocks.Plugin).On("GetNodeByID", nm.ctx, uuidMatches(id)).Return(&fftypes.Node{ID: id}, nil)
	res, err := nm.GetNodeByID(nm.ctx, id.String())
	assert.NoError(t, err)
	assert.Equal(t, *id, *res.ID)
}

func TestGetNodeByIDBadUUID(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	_, err := nm.GetNodeByID(nm.ctx, "bad")
	assert.Regexp(t, "FF10142", err)
}

func TestGetOrganizations(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	nm.database.(*databasemocks.Plugin).On("GetOrganizations", nm.ctx, mock.Anything).Return([]*fftypes.Organization{}, nil)
	res, err := nm.GetOrganizations(nm.ctx, database.OrganizationQueryFactory.NewFilter(nm.ctx).And())
	assert.NoError(t, err)
	assert.Empty(t, res)
}

func TestGetNodes(t *testing.T) {
	nm, cancel := newTestNetworkmap(t)
	defer cancel()
	nm.database.(*databasemocks.Plugin).On("GetNodes", nm.ctx, mock.Anything).Return([]*fftypes.Node{}, nil)
	res, err := nm.GetNodes(nm.ctx, database.NodeQueryFactory.NewFilter(nm.ctx).And())
	assert.NoError(t, err)
	assert.Empty(t, res)
}
