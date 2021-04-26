// Copyright © 2021 Kaleido, Inc.
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

	"github.com/kaleido-io/firefly/internal/i18n"
)

var putDefDatatype = Route{

	Name:            "putDefDatatype",
	Path:            "defs/datatypes",
	Method:          http.MethodPut,
	PathParams:      nil,
	QueryParams:     nil,
	Description:     i18n.MsgPutDefDataTypeDesc,
	JSONInputValue:  func() interface{} { return &PutDefDatatypeInput{} },
	JSONOutputValue: func() interface{} { return &PutDefDatatypeOutput{} },
	// TODO: Move to real implementation
	JSONHandler: func(req *http.Request, input interface{}, output interface{}) (status int, err error) {
		return 200, nil
	},
}

type PutDefDatatypeInput struct {
}

type PutDefDatatypeOutput struct {
}
