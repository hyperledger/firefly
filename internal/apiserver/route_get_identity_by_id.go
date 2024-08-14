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

package apiserver

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hyperledger/firefly-common/pkg/ffapi"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

var getIdentityByID = &ffapi.Route{
	Name:   "getIdentityByID",
	Path:   "identities/{id:.+}",
	Method: http.MethodGet,
	PathParams: []*ffapi.PathParam{
		{Name: "id", Example: "id", Description: coremsgs.APIParamsIdentityIDs},
	},
	QueryParams: []*ffapi.QueryParam{
		{Name: "fetchverifiers", Example: "true", Description: coremsgs.APIParamsFetchVerifiers, IsBool: true},
	},
	Description:     coremsgs.APIEndpointsGetIdentityByID,
	JSONInputValue:  nil,
	JSONOutputValue: func() interface{} { return &core.Identity{} },
	JSONOutputCodes: []int{http.StatusOK},
	Extensions: &coreExtensions{
		CoreJSONHandler: func(r *ffapi.APIRequest, cr *coreRequest) (output interface{}, err error) {
			switch id := r.PP["id"]; {
			case isUUID(id):
				if strings.EqualFold(r.QP["fetchverifiers"], "true") {
					return cr.or.NetworkMap().GetIdentityByIDWithVerifiers(cr.ctx, id)
				}
				return cr.or.NetworkMap().GetIdentityByID(cr.ctx, id)
			case isDID(id):
				if strings.EqualFold(r.QP["fetchverifiers"], "true") {
					return cr.or.NetworkMap().GetIdentityByDIDWithVerifiers(cr.ctx, id)
				}
				return cr.or.NetworkMap().GetIdentityByDID(cr.ctx, id)
			}
			return nil, i18n.NewError(cr.ctx, i18n.MsgUnknownIdentityType)
		},
	},
}

// isUUID checks if the given string is a UUID
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// isDID checks if the given string is a potential DID
func isDID(s string) bool {
	return strings.HasPrefix(s, "did:")
}
