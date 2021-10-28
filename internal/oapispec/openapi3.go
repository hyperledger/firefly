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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func SwaggerGen(ctx context.Context, routes []*Route, url string) *openapi3.T {

	doc := &openapi3.T{
		OpenAPI: "3.0.2",
		Servers: openapi3.Servers{
			{URL: url + "/api/v1"},
		},
		Info: &openapi3.Info{
			Title:       "FireFly",
			Version:     "1.0",
			Description: "Copyright © 2021 Kaleido, Inc.",
		},
	}
	opIds := make(map[string]bool)
	for _, route := range routes {
		if route.Name == "" || opIds[route.Name] {
			log.Panicf("Duplicate/invalid name (used as operation ID in swagger): %s", route.Name)
		}
		addRoute(ctx, doc, route)
		opIds[route.Name] = true
	}
	return doc
}

func getPathItem(doc *openapi3.T, path string) *openapi3.PathItem {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if doc.Paths == nil {
		doc.Paths = openapi3.Paths{}
	}
	pi, ok := doc.Paths[path]
	if ok {
		return pi
	}
	pi = &openapi3.PathItem{}
	doc.Paths[path] = pi
	return pi
}

func initInput(op *openapi3.Operation) {
	op.RequestBody = &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Content: openapi3.Content{},
		},
	}
}

func ffTagHandler(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
	if ffEnum := tag.Get("ffenum"); ffEnum != "" {
		schema.Enum = fftypes.FFEnumValues(ffEnum)
	}
	return nil
}

func addInput(ctx context.Context, input interface{}, mask []string, schemaDef func(context.Context) string, op *openapi3.Operation) {
	var schemaRef *openapi3.SchemaRef
	if schemaDef != nil {
		err := json.Unmarshal([]byte(schemaDef(ctx)), &schemaRef)
		if err != nil {
			panic(fmt.Sprintf("invalid schema for %T: %s", input, err))
		}
	}
	if schemaRef == nil {
		schemaRef, _, _ = openapi3gen.NewSchemaRefForValue(maskFields(input, mask), openapi3gen.SchemaCustomizer(ffTagHandler))
	}
	op.RequestBody.Value.Content["application/json"] = &openapi3.MediaType{
		Schema: schemaRef,
	}
}

func addFormInput(ctx context.Context, op *openapi3.Operation, formParams []*FormParam) {
	props := openapi3.Schemas{
		"filename.ext": &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:   "string",
				Format: "binary",
			},
		},
	}
	for _, fp := range formParams {
		props[fp.Name] = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Description: i18n.Expand(ctx, i18n.MsgSuccessResponse),
				Type:        "string",
			},
		}
	}

	op.RequestBody.Value.Content["multipart/form-data"] = &openapi3.MediaType{
		Schema: &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:       "object",
				Properties: props,
			},
		},
	}
}

func addOutput(ctx context.Context, route *Route, output interface{}, op *openapi3.Operation) {
	schemaRef, _, _ := openapi3gen.NewSchemaRefForValue(output)
	s := i18n.Expand(ctx, i18n.MsgSuccessResponse)
	for _, code := range route.JSONOutputCodes {
		op.Responses[strconv.FormatInt(int64(code), 10)] = &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: &s,
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: schemaRef,
					},
				},
			},
		}
	}
}

func addParam(ctx context.Context, op *openapi3.Operation, in, name, def, example string, description i18n.MessageKey, deprecated bool, msgArgs ...interface{}) {
	required := false
	if in == "path" {
		required = true
	}
	var defValue interface{}
	if def != "" {
		defValue = &def
	}
	var exampleValue interface{}
	if example != "" {
		exampleValue = example
	}
	op.Parameters = append(op.Parameters, &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			In:          in,
			Name:        name,
			Required:    required,
			Deprecated:  deprecated,
			Description: i18n.Expand(ctx, description, msgArgs...),
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    "string",
					Default: defValue,
					Example: exampleValue,
				},
			},
		},
	})
}

func addRoute(ctx context.Context, doc *openapi3.T, route *Route) {
	pi := getPathItem(doc, route.Path)
	op := &openapi3.Operation{
		Description: i18n.Expand(ctx, route.Description),
		OperationID: route.Name,
		Responses:   openapi3.NewResponses(),
		Deprecated:  route.Deprecated,
	}
	if route.Method != http.MethodGet && route.Method != http.MethodDelete {
		var input interface{}
		if route.JSONInputValue != nil {
			input = route.JSONInputValue()
		}
		initInput(op)
		if input != nil {
			addInput(ctx, input, route.JSONInputMask, route.JSONInputSchema, op)
		}
		if route.FormUploadHandler != nil {
			addFormInput(ctx, op, route.FormParams)
		}
	}
	var output interface{}
	if route.JSONOutputValue != nil {
		output = route.JSONOutputValue()
	}
	if output != nil {
		addOutput(ctx, route, output, op)
	}
	for _, p := range route.PathParams {
		example := p.Example
		if p.ExampleFromConf != "" {
			example = config.GetString(p.ExampleFromConf)
		}
		addParam(ctx, op, "path", p.Name, p.Default, example, p.Description, false)
	}
	for _, q := range route.QueryParams {
		example := q.Example
		if q.ExampleFromConf != "" {
			example = config.GetString(q.ExampleFromConf)
		}
		addParam(ctx, op, "query", q.Name, q.Default, example, q.Description, q.Deprecated)
	}
	addParam(ctx, op, "header", "Request-Timeout", config.GetString(config.APIRequestTimeout), "", i18n.MsgRequestTimeoutDesc, false)
	if route.FilterFactory != nil {
		fields := route.FilterFactory.NewFilter(ctx).Fields()
		sort.Strings(fields)
		for _, field := range fields {
			addParam(ctx, op, "query", field, "", "", i18n.MsgFilterParamDesc, false)
		}
		addParam(ctx, op, "query", "sort", "", "", i18n.MsgFilterSortDesc, false)
		addParam(ctx, op, "query", "ascending", "", "", i18n.MsgFilterAscendingDesc, false)
		addParam(ctx, op, "query", "descending", "", "", i18n.MsgFilterDescendingDesc, false)
		addParam(ctx, op, "query", "skip", "", "", i18n.MsgFilterSkipDesc, false, config.GetUint(config.APIMaxFilterSkip))
		addParam(ctx, op, "query", "limit", "", config.GetString(config.APIDefaultFilterLimit), i18n.MsgFilterLimitDesc, false, config.GetUint(config.APIMaxFilterLimit))
		addParam(ctx, op, "query", "count", "", "", i18n.MsgFilterCountDesc, false)
	}
	switch route.Method {
	case http.MethodGet:
		pi.Get = op
	case http.MethodPut:
		pi.Put = op
	case http.MethodPost:
		pi.Post = op
	case http.MethodDelete:
		pi.Delete = op
	}
}

func maskFieldsOnStruct(t reflect.Type, mask []string) reflect.Type {
	fieldCount := t.NumField()
	newFields := make([]reflect.StructField, fieldCount)
	for i := 0; i < fieldCount; i++ {
		field := t.FieldByIndex([]int{i})
		for _, m := range mask {
			if strings.EqualFold(field.Name, m) {
				field.Tag = "`json:-`"
			}
		}
		newFields[i] = field
	}
	return reflect.StructOf(newFields)
}

func maskFields(input interface{}, mask []string) interface{} {
	t := reflect.TypeOf(input)
	newStruct := maskFieldsOnStruct(t.Elem(), mask)
	i := reflect.New(newStruct).Interface()
	return i
}
