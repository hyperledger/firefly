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

package oapispec

import (
	"context"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
)

// Route defines each API operation on the REST API of Firefly
// Having a standard pluggable layer here on top of Gorilla allows us to autmoatically
// maintain the OpenAPI specification in-line with the code, while retaining the
// power of the Gorilla mux without a deep abstraction layer.
type Route struct {
	// Name is the operation name that will go into the Swagger definition, and prefix input/output schema
	Name string
	// Path is a Gorilla mux path spec
	Path string
	// PathParams is a list of documented path parameters
	PathParams []*PathParam
	// QueryParams is a list of documented query parameters
	QueryParams []*QueryParam
	// FormParams is a list of documented multi-part form parameters - combine with FormUploadHandler
	FormParams []*FormParam
	// FilterFactory is a reference to a filter object that defines the search param on resource collection interfaces
	FilterFactory database.QueryFactory
	// Method is the HTTP method
	Method string
	// Description is a message key to a translatable description of the operation
	Description i18n.MessageKey
	// JSONInputValue is a function that returns a pointer to a structure to take JSON input
	JSONInputValue func() interface{}
	// JSONInputMask are fields that aren't available for users to supply on input
	JSONInputMask []string
	// JSONInputSchema is a custom schema definition, for the case where the auto-gen + mask isn't good enough
	JSONInputSchema func(ctx context.Context) string
	// JSONOutputSchema is a custom schema definition, for the case where the auto-gen + mask isn't good enough
	JSONOutputSchema func(ctx context.Context) string
	// JSONOutputValue is a function that returns a pointer to a structure to take JSON output
	JSONOutputValue func() interface{}
	// JSONOutputCodes is the success response code
	JSONOutputCodes []int
	// JSONHandler is a function for handling JSON content type input. Input/Ouptut objects are returned by JSONInputValue/JSONOutputValue funcs
	JSONHandler func(r *APIRequest) (output interface{}, err error)
	// FormUploadHandler takes a single file upload, and returns a JSON object
	FormUploadHandler func(r *APIRequest) (output interface{}, err error)
	// Deprecated whether this route is deprecated
	Deprecated bool
}

// PathParam is a description of a path parameter
type PathParam struct {
	// Name is the name of the parameter, from the Gorilla path mux
	Name string
	// Default is the value that will be used in the case no value is supplied
	Default string
	// Example is a field to fill in, in the helper UI
	Example string
	// ExampleFromConf is a field to fill in, in the helper UI, from the runtime configuration
	ExampleFromConf config.RootKey
	// Description is a message key to a translatable description of the parameter
	Description i18n.MessageKey
}

// QueryParam is a description of a path parameter
type QueryParam struct {
	// Name is the name of the parameter, from the Gorilla path mux
	Name string
	// IsBool if this is a boolean query
	IsBool bool
	// Default is the value that will be used in the case no value is supplied
	Default string
	// Example is a field to fill in, in the helper UI
	Example string
	// ExampleFromConf is a field to fill in, in the helper UI, from the runtime configuration
	ExampleFromConf config.RootKey
	// Description is a message key to a translatable description of the parameter
	Description i18n.MessageKey
	// Deprecated whether this param is deprecated
	Deprecated bool
}

// FormParam is a description of a multi-part form parameter
type FormParam struct {
	// Name is the name of the parameter, from the Gorilla path mux
	Name string
	// Description is a message key to a translatable description of the parameter
	Description i18n.MessageKey
}
