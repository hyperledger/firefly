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

package apiserver

import (
	"context"
	"net/http"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var postRequestMessage = &oapispec.Route{
	Name:   "postRequestMessage",
	Path:   "namespaces/{ns}/request/message",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.MessageInOut{} },
	JSONInputSchema: func(ctx context.Context) string { return privateSendSchema },
	JSONOutputValue: func() interface{} { return &fftypes.MessageInOut{} },
	JSONOutputCodes: []int{http.StatusOK}, // Sync operation
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		msg := r.Input.(*fftypes.MessageInOut)
		if msg.Header.Group == nil && (msg.Group == nil || len(msg.Group.Members) == 0) {
			return nil, i18n.NewError(r.Ctx, i18n.MsgRequestMustBePrivate)
		}
		output, err = r.Or.PrivateMessaging().RequestReply(r.Ctx, r.PP["ns"], msg)
		return output, err
	},
	Deprecated: true, // moving to more intutitive route/return structure
}
