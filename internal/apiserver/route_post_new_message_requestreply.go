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

	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/core"
)

var postNewMessageRequestReply = &oapispec.Route{
	Name:            "postNewMessageRequestReply",
	Path:            "messages/requestreply",
	Method:          http.MethodPost,
	PathParams:      nil,
	QueryParams:     nil,
	FilterFactory:   nil,
	DescriptionKey:  coremsgs.APIEndpointsPostNewMessageRequestReply,
	JSONInputValue:  func() interface{} { return &core.MessageInOut{} },
	JSONOutputValue: func() interface{} { return &core.MessageInOut{} },
	JSONOutputCodes: []int{http.StatusOK}, // Sync operation
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		output, err = getOr(r.Ctx).RequestReply(r.Ctx, extractNamespace(r.PP), r.Input.(*core.MessageInOut))
		return output, err
	},
}
