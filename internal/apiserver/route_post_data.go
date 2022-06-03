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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/core"
)

var postData = &oapispec.Route{
	Name:        "postData",
	Path:        "data",
	Method:      http.MethodPost,
	PathParams:  nil,
	QueryParams: nil,
	FormParams: []*oapispec.FormParam{
		{Name: "autometa", Description: coremsgs.APIParamsAutometa},
		{Name: "metadata", Description: coremsgs.APIParamsMetadata},
		{Name: "validator", Description: coremsgs.APIParamsValidator},
		{Name: "datatype.name", Description: coremsgs.APIParamsDatatypeName},
		{Name: "datatype.version", Description: coremsgs.APIParamsDatatypeVersion},
	},
	FilterFactory:   nil,
	Description:     coremsgs.APIEndpointsPostData,
	JSONInputValue:  func() interface{} { return &core.DataRefOrValue{} },
	JSONOutputValue: func() interface{} { return &core.Data{} },
	JSONOutputCodes: []int{http.StatusCreated},
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		ns := extractNamespace(r.PP)
		output, err = getOr(r.Ctx, ns).Data().UploadJSON(r.Ctx, ns, r.Input.(*core.DataRefOrValue))
		return output, err
	},
	FormUploadHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		data := &core.DataRefOrValue{}
		validator := r.FP["validator"]
		if len(validator) > 0 {
			data.Validator = core.ValidatorType(validator)
		}
		if r.FP["datatype.name"] != "" {
			data.Datatype = &core.DatatypeRef{
				Name:    r.FP["datatype.name"],
				Version: r.FP["datatype.version"],
			}
		}
		metadata := r.FP["metadata"]
		if len(metadata) > 0 {
			// The metadata might be JSON, or just a simple string. Try to unmarshal and see
			var marshalCheck interface{}
			if err := json.Unmarshal([]byte(metadata), &marshalCheck); err != nil {
				metadata = fmt.Sprintf(`"%s"`, metadata)
			}
			data.Value = fftypes.JSONAnyPtr(metadata)
		}
		ns := extractNamespace(r.PP)
		output, err = getOr(r.Ctx, ns).Data().UploadBlob(r.Ctx, ns, data, r.Part, strings.EqualFold(r.FP["autometa"], "true"))
		return output, err
	},
}
