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

	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var postResetConfig = &oapispec.Route{
	Name:            "postResetConfig",
	Path:            "config/reset",
	Method:          http.MethodPost,
	PathParams:      nil,
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.Byteable{} },
	JSONOutputValue: nil,
	JSONOutputCode:  http.StatusNoContent,
	JSONInputSchema: func(ctx context.Context) string { return emptyObjectSchema },
	JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		r.Or.ResetConfig(r.Ctx)
		return nil, nil
	},
}
