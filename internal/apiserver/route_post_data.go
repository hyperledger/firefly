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
	"net/http"

	"github.com/hyperledger-labs/firefly/internal/config"
	"github.com/hyperledger-labs/firefly/internal/i18n"
	"github.com/hyperledger-labs/firefly/internal/oapispec"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
)

var postData = &oapispec.Route{
	Name:   "postData",
	Path:   "namespaces/{ns}/data",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.Data{} },
	JSONInputMask:   []string{"ID", "Namespace", "Created", "Blob", "Blobstore", "Hash"},
	JSONOutputValue: func() interface{} { return &fftypes.Data{} },
	JSONOutputCode:  http.StatusCreated, // Sync operation
	JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		output, err = r.Or.Data().UploadJSON(r.Ctx, r.PP["ns"], r.Input.(*fftypes.Data))
		return output, err
	},
	FormUploadHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		output, err = r.Or.Data().UploadBLOB(r.Ctx, r.PP["ns"], r.FReader)
		return output, err
	},
}
