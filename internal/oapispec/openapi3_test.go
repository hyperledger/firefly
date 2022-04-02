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

package oapispec

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/database"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
)

var testRoutes = []*Route{
	{
		Name:   "op1",
		Path:   "namespaces/{ns}/example1/{id}",
		Method: http.MethodPost,
		PathParams: []*PathParam{
			{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
			{Name: "id", Description: i18n.MsgTBD},
		},
		QueryParams:     nil,
		FilterFactory:   nil,
		Description:     i18n.MsgTBD,
		JSONInputValue:  func() interface{} { return &fftypes.MessageInOut{} },
		JSONOutputValue: func() interface{} { return &fftypes.Batch{} },
		JSONOutputCodes: []int{http.StatusOK},
	},
	{
		Name:            "op2",
		Path:            "example2",
		Method:          http.MethodGet,
		PathParams:      nil,
		QueryParams:     nil,
		FilterFactory:   database.MessageQueryFactory,
		Description:     i18n.MsgTBD,
		JSONInputValue:  func() interface{} { return nil },
		JSONOutputCodes: []int{http.StatusOK},
	},
	{
		Name:       "op3",
		Path:       "example2",
		Method:     http.MethodPut,
		PathParams: nil,
		QueryParams: []*QueryParam{
			{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
			{Name: "id", Description: i18n.MsgTBD},
			{Name: "myfield", Default: "val1", Description: i18n.MsgTBD},
		},
		FilterFactory:   nil,
		Description:     i18n.MsgTBD,
		JSONInputValue:  func() interface{} { return &fftypes.MessageInOut{} },
		JSONOutputValue: func() interface{} { return nil },
		JSONOutputCodes: []int{http.StatusNoContent},
		FormParams: []*FormParam{
			{Name: "metadata", Description: i18n.MsgTBD},
		},
		FormUploadHandler: func(r *APIRequest) (output interface{}, err error) { return nil, nil },
	},
	{
		Name:   "op4",
		Path:   "example2/{id}",
		Method: http.MethodDelete,
		PathParams: []*PathParam{
			{Name: "id", Description: i18n.MsgTBD},
		},
		QueryParams:     nil,
		FilterFactory:   nil,
		Description:     i18n.MsgTBD,
		JSONInputValue:  func() interface{} { return nil },
		JSONOutputValue: func() interface{} { return nil },
		JSONOutputCodes: []int{http.StatusNoContent},
	},
	{
		Name:            "op5",
		Path:            "example2",
		Method:          http.MethodPost,
		PathParams:      nil,
		QueryParams:     nil,
		FilterFactory:   nil,
		Description:     i18n.MsgTBD,
		JSONInputValue:  func() interface{} { return &fftypes.Data{} },
		JSONOutputValue: func() interface{} { return &fftypes.Data{} },
		JSONOutputCodes: []int{http.StatusOK},
	},
}

type TestInOutType struct {
	Length           float64 `ffstruct:"TestInOutType" json:"length"`
	Width            float64 `ffstruct:"TestInOutType" json:"width"`
	Height           float64 `ffstruct:"TestInOutType" json:"height"`
	Volume           float64 `ffstruct:"TestInOutType" json:"volume" ffexcludeinput:"true"`
	Secret           string  `ffstruct:"TestInOutType" json:"secret" ffexclude:"true"`
	Conditional      string  `ffstruct:"TestInOutType" json:"conditional" ffexclude:"PostTagTest"`
	ConditionalInput string  `ffstruct:"TestInOutType" json:"conditionalInput" ffexcludeinput:"PostTagTest"`
}

type TestNonTaggedType struct {
	NoFFStructTag string `json:"noFFStructTag"`
}

func TestOpenAPI3SwaggerGen(t *testing.T) {

	config.Reset()

	doc := SwaggerGen(context.Background(), testRoutes, &SwaggerGenConfig{
		Title:   "UnitTest",
		Version: "1.0",
		BaseURL: "http://localhost:12345/api/v1",
	})
	err := doc.Validate(context.Background())
	assert.NoError(t, err)

	b, err := yaml.Marshal(doc)
	assert.NoError(t, err)
	fmt.Print(string(b))
}

func TestBadCustomInputSchema(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:             "op6",
			Path:             "namespaces/{ns}/example1/{id}",
			Method:           http.MethodPost,
			JSONInputValue:   func() interface{} { return &fftypes.Message{} },
			JSONInputMask:    []string{"id"},
			JSONOutputCodes:  []int{http.StatusOK},
			JSONInputSchema:  func(ctx context.Context) string { return `!json` },
			JSONOutputSchema: func(ctx context.Context) string { return `!json` },
		},
	}
	assert.PanicsWithValue(t, "invalid schema: invalid character '!' looking for beginning of value", func() {
		_ = SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
			Title:   "UnitTest",
			Version: "1.0",
			BaseURL: "http://localhost:12345/api/v1",
		})
	})
}

func TestBadCustomOutputSchema(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:            "op7",
			Path:            "namespaces/{ns}/example1/{id}",
			Method:          http.MethodGet,
			JSONInputValue:  func() interface{} { return &fftypes.Message{} },
			JSONInputMask:   []string{"id"},
			JSONOutputCodes: []int{http.StatusOK}, JSONInputSchema: func(ctx context.Context) string { return `!json` },
			JSONOutputSchema: func(ctx context.Context) string { return `!json` },
		},
	}
	assert.PanicsWithValue(t, "invalid schema: invalid character '!' looking for beginning of value", func() {
		_ = SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
			Title:   "UnitTest",
			Version: "1.0",
			BaseURL: "http://localhost:12345/api/v1",
		})
	})
}

func TestDuplicateOperationIDCheck(t *testing.T) {
	routes := []*Route{
		{Name: "op1"}, {Name: "op1"},
	}
	assert.PanicsWithValue(t, "Duplicate/invalid name (used as operation ID in swagger): op1", func() {
		_ = SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
			Title:   "UnitTest",
			Version: "1.0",
			BaseURL: "http://localhost:12345/api/v1",
		})
	})
}

func TestWildcards(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:            "op1",
			Path:            "namespaces/{ns}/example1/{id:.*wildcard.*}",
			Method:          http.MethodPost,
			JSONInputValue:  func() interface{} { return &fftypes.Message{} },
			JSONOutputCodes: []int{http.StatusOK},
		},
	}
	swagger := SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
		Title:   "UnitTest",
		Version: "1.0",
		BaseURL: "http://localhost:12345/api/v1",
	})
	assert.NotNil(t, swagger.Paths["/namespaces/{ns}/example1/{id}"])
}

func TestFFExcludeTag(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:            "PostTagTest",
			Path:            "namespaces/{ns}/example1/test",
			Method:          http.MethodPost,
			JSONInputValue:  func() interface{} { return &TestInOutType{} },
			JSONOutputValue: func() interface{} { return &TestInOutType{} },
			JSONOutputCodes: []int{http.StatusOK},
		},
	}
	swagger := SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
		Title:   "UnitTest",
		Version: "1.0",
		BaseURL: "http://localhost:12345/api/v1",
	})
	assert.NotNil(t, swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value)
	length, err := swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value.Properties.JSONLookup("length")
	assert.NoError(t, err)
	assert.NotNil(t, length)
	width, err := swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value.Properties.JSONLookup("width")
	assert.NoError(t, err)
	assert.NotNil(t, width)
	_, err = swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value.Properties.JSONLookup("secret")
	assert.Regexp(t, "object has no field", err)
	_, err = swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value.Properties.JSONLookup("conditional")
	assert.Regexp(t, "object has no field", err)
	_, err = swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value.Properties.JSONLookup("conditionalInput")
	assert.Regexp(t, "object has no field", err)
}

func TestCustomSchema(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:   "PostCustomSchema",
			Path:   "namespaces/{ns}/example1/test",
			Method: http.MethodPost,
			JSONInputSchema: func(ctx context.Context) string {
				return `{"properties": {"foo": {"type": "string", "description": "a custom foo"}}}`
			},
			JSONOutputSchema: func(ctx context.Context) string {
				return `{"properties": {"bar": {"type": "string", "description": "a custom bar"}}}`
			},
		},
	}
	swagger := SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
		Title:   "UnitTest",
		Version: "1.0",
		BaseURL: "http://localhost:12345/api/v1",
	})
	assert.NotNil(t, swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value)
	length, err := swagger.Paths["/namespaces/{ns}/example1/test"].Post.RequestBody.Value.Content.Get("application/json").Schema.Value.Properties.JSONLookup("foo")
	assert.NoError(t, err)
	assert.NotNil(t, length)
}

func TestPanicOnMissingDescription(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:            "PostPanicOnMissingDescription",
			Path:            "namespaces/{ns}/example1/test",
			Method:          http.MethodPost,
			JSONInputValue:  func() interface{} { return &TestInOutType{} },
			JSONOutputValue: func() interface{} { return &TestInOutType{} },
			JSONOutputCodes: []int{http.StatusOK},
		},
	}
	assert.PanicsWithValue(t, "invalid schema: FF10381: Field description missing for 'TestInOutType.conditional' on route 'PostPanicOnMissingDescription'", func() {
		_ = SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
			Title:                     "UnitTest",
			Version:                   "1.0",
			BaseURL:                   "http://localhost:12345/api/v1",
			PanicOnMissingDescription: true,
		})
	})
}

func TestPanicOnMissingFFStructTag(t *testing.T) {
	config.Reset()
	routes := []*Route{
		{
			Name:            "GetPanicOnMissingFFStructTag",
			Path:            "namespaces/{ns}/example1/test",
			Method:          http.MethodGet,
			JSONOutputValue: func() interface{} { return &TestNonTaggedType{} },
			JSONOutputCodes: []int{http.StatusOK},
		},
	}
	assert.PanicsWithValue(t, "invalid schema: FF10382: ffstruct tag is missing for 'noFFStructTag' on route 'GetPanicOnMissingFFStructTag'", func() {
		_ = SwaggerGen(context.Background(), routes, &SwaggerGenConfig{
			Title:                     "UnitTest",
			Version:                   "1.0",
			BaseURL:                   "http://localhost:12345/api/v1",
			PanicOnMissingDescription: true,
		})
	})
}
