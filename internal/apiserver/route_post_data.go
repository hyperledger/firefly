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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	QueryParams: nil,
	FormParams: []*oapispec.FormParam{
		{Name: "autometa", Description: i18n.MsgTBD},
		{Name: "metadata", Description: i18n.MsgTBD},
		{Name: "validator", Description: i18n.MsgTBD},
		{Name: "datatype.name", Description: i18n.MsgTBD},
		{Name: "datatype.version", Description: i18n.MsgTBD},
	},
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.DataRefOrValue{} },
	JSONInputMask:   nil,
	JSONOutputValue: func() interface{} { return &fftypes.Data{} },
	JSONOutputCode:  http.StatusCreated, // Sync operation
	JSONHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		output, err = r.Or.Data().UploadJSON(r.Ctx, r.PP["ns"], r.Input.(*fftypes.DataRefOrValue))
		return output, err
	},
	FormUploadHandler: func(r oapispec.APIRequest) (output interface{}, err error) {
		data := &fftypes.DataRefOrValue{}
		metadata := r.FP["metadata"]
		if len(metadata) > 0 {
			// The metadata might be JSON, or just a simple string. Try to unmarshal and see
			var marshalCheck interface{}
			if err := json.Unmarshal([]byte(metadata), &marshalCheck); err != nil {
				metadata = fmt.Sprintf(`"%s"`, metadata)
			}
			data.Value = fftypes.Byteable(metadata)
			data.Validator = fftypes.ValidatorType(r.FP["validator"])
			if r.FP["datatype.name"] != "" {
				data.Datatype = &fftypes.DatatypeRef{
					Name:    r.FP["datatype.name"],
					Version: r.FP["datatype.version"],
				}
			}
		}
		output, err = r.Or.Data().UploadBLOB(r.Ctx, r.PP["ns"], data, r.Part, strings.EqualFold(r.FP["autometa"], "true"))
		return output, err
	},
}
