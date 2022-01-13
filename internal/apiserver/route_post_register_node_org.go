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
	"context"
	"net/http"

	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var postRegisterNodeOrg = &oapispec.Route{
	Name:            "postRegisterNodeOrg",
	Path:            "network/register/node/organization",
	Method:          http.MethodPost,
	PathParams:      nil,
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.EmptyInput{} },
	JSONInputMask:   nil,
	JSONInputSchema: func(ctx context.Context) string { return emptyObjectSchema },
	JSONOutputValue: func() interface{} { return &fftypes.Message{} },
	JSONOutputCodes: []int{http.StatusAccepted}, // Async operation
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		// This (old) route is always async, and returns the message
		_, output, err = getOr(r.Ctx).NetworkMap().RegisterNodeOrganization(r.Ctx, false)
		return output, err
	},
	Deprecated: true, // moving to more intutitive route/return structure
}
