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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/internal/coremsgs"
	"github.com/hyperledger/firefly/pkg/core"
)

type SwaggerGenConfig struct {
	BaseURL                   string
	Title                     string
	Version                   string
	Description               string
	PanicOnMissingDescription bool
}

var customRegexRemoval = regexp.MustCompile(`{(\w+)\:[^}]+}`)

func SwaggerGen(ctx context.Context, routes []*Route, conf *SwaggerGenConfig) *openapi3.T {

	doc := &openapi3.T{
		OpenAPI: "3.0.2",
		Servers: openapi3.Servers{
			{URL: conf.BaseURL},
		},
		Info: &openapi3.Info{
			Title:       conf.Title,
			Version:     conf.Version,
			Description: conf.Description,
		},
		Components: openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}
	opIds := make(map[string]bool)
	for _, route := range routes {
		if route.Name == "" || opIds[route.Name] {
			log.Panicf("Duplicate/invalid name (used as operation ID in swagger): %s", route.Name)
		}
		addRoute(ctx, doc, route, conf)
		opIds[route.Name] = true
	}
	return doc
}

func getPathItem(doc *openapi3.T, path string) *openapi3.PathItem {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = customRegexRemoval.ReplaceAllString(path, `{$1}`)
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

func isTrue(str string) bool {
	return strings.EqualFold(str, "true")
}

func ffInputTagHandler(ctx context.Context, route *Route, name string, tag reflect.StructTag, schema *openapi3.Schema, conf *SwaggerGenConfig) error {
	if isTrue(tag.Get("ffexcludeinput")) {
		return &openapi3gen.ExcludeSchemaSentinel{}
	}
	if taggedRoutes, ok := tag.Lookup("ffexcludeinput"); ok {
		for _, r := range strings.Split(taggedRoutes, ",") {
			if route.Name == r {
				return &openapi3gen.ExcludeSchemaSentinel{}
			}
		}
	}
	return ffTagHandler(ctx, route, name, tag, schema, conf)
}

func ffOutputTagHandler(ctx context.Context, route *Route, name string, tag reflect.StructTag, schema *openapi3.Schema, conf *SwaggerGenConfig) error {
	if isTrue(tag.Get("ffexcludeoutput")) {
		return &openapi3gen.ExcludeSchemaSentinel{}
	}
	return ffTagHandler(ctx, route, name, tag, schema, conf)
}

func ffTagHandler(ctx context.Context, route *Route, name string, tag reflect.StructTag, schema *openapi3.Schema, conf *SwaggerGenConfig) error {
	if ffEnum := tag.Get("ffenum"); ffEnum != "" {
		schema.Enum = core.FFEnumValues(ffEnum)
	}
	if isTrue(tag.Get("ffexclude")) {
		return &openapi3gen.ExcludeSchemaSentinel{}
	}
	if taggedRoutes, ok := tag.Lookup("ffexclude"); ok {
		for _, r := range strings.Split(taggedRoutes, ",") {
			if route.Name == r {
				return &openapi3gen.ExcludeSchemaSentinel{}
			}
		}
	}
	if name != "_root" {
		if structName, ok := tag.Lookup("ffstruct"); ok {
			key := fmt.Sprintf("%s.%s", structName, name)
			description := i18n.Expand(ctx, i18n.MessageKey(key))
			if description == key && conf.PanicOnMissingDescription {
				return i18n.NewError(ctx, coremsgs.MsgFieldDescriptionMissing, key, route.Name)
			}
			schema.Description = description
		} else if conf.PanicOnMissingDescription {
			return i18n.NewError(ctx, coremsgs.MsgFFStructTagMissing, name, route.Name)
		}
	}
	return nil
}

func addCustomType(t reflect.Type, schema *openapi3.Schema) {
	typeString := "string"
	switch t.Name() {
	case "UUID":
		schema.Type = typeString
		schema.Format = "uuid"
	case "FFTime":
		schema.Type = typeString
		schema.Format = "date-time"
	case "Bytes32":
		schema.Type = typeString
		schema.Format = "byte"
	case "FFBigInt":
		schema.Type = typeString
	case "JSONAny":
		schema.Type = ""
	}
}

func addInput(ctx context.Context, doc *openapi3.T, route *Route, op *openapi3.Operation, conf *SwaggerGenConfig) {
	var schemaRef *openapi3.SchemaRef
	var err error
	schemaCustomizer := func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		addCustomType(t, schema)
		return ffInputTagHandler(ctx, route, name, tag, schema, conf)
	}
	switch {
	case route.JSONInputSchema != nil:
		err = json.Unmarshal([]byte(route.JSONInputSchema(ctx)), &schemaRef)
		if err != nil {
			panic(fmt.Sprintf("invalid schema: %s", err))
		}
	case route.JSONInputValue != nil:
		schemaRef, err = openapi3gen.NewSchemaRefForValue(route.JSONInputValue(), doc.Components.Schemas, openapi3gen.SchemaCustomizer(schemaCustomizer))
		if err != nil {
			panic(fmt.Sprintf("invalid schema: %s", err))
		}
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
				Description: i18n.Expand(ctx, coremsgs.APISuccessResponse),
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

func addOutput(ctx context.Context, doc *openapi3.T, route *Route, op *openapi3.Operation, conf *SwaggerGenConfig) {
	var schemaRef *openapi3.SchemaRef
	var err error
	s := i18n.Expand(ctx, coremsgs.APISuccessResponse)
	schemaCustomizer := func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		addCustomType(t, schema)
		return ffOutputTagHandler(ctx, route, name, tag, schema, conf)
	}
	switch {
	case route.JSONOutputSchema != nil:
		err := json.Unmarshal([]byte(route.JSONOutputSchema(ctx)), &schemaRef)
		if err != nil {
			panic(fmt.Sprintf("invalid schema: %s", err))
		}
	case route.JSONOutputValue != nil:
		outputValue := route.JSONOutputValue()
		if outputValue != nil {
			schemaRef, err = openapi3gen.NewSchemaRefForValue(outputValue, doc.Components.Schemas, openapi3gen.SchemaCustomizer(schemaCustomizer))
			if err != nil {
				panic(fmt.Sprintf("invalid schema: %s", err))
			}
		}
	}
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

func addRoute(ctx context.Context, doc *openapi3.T, route *Route, conf *SwaggerGenConfig) {
	pi := getPathItem(doc, route.Path)
	routeDescription := i18n.Expand(ctx, route.Description)
	if routeDescription == "" && conf.PanicOnMissingDescription {
		log.Panicf(i18n.NewError(ctx, coremsgs.MsgRouteDescriptionMissing, route.Name).Error())
	}
	op := &openapi3.Operation{
		Description: routeDescription,
		OperationID: route.Name,
		Responses:   openapi3.NewResponses(),
		Deprecated:  route.Deprecated,
		Tags:        []string{route.Tag},
	}
	if route.Method != http.MethodGet && route.Method != http.MethodDelete {
		initInput(op)
		addInput(ctx, doc, route, op, conf)
		if route.FormUploadHandler != nil {
			addFormInput(ctx, op, route.FormParams)
		}
	}
	addOutput(ctx, doc, route, op, conf)
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
	addParam(ctx, op, "header", "Request-Timeout", config.GetString(coreconfig.APIRequestTimeout), "", coremsgs.APIRequestTimeoutDesc, false)
	if route.FilterFactory != nil {
		fields := route.FilterFactory.NewFilter(ctx).Fields()
		sort.Strings(fields)
		for _, field := range fields {
			addParam(ctx, op, "query", field, "", "", coremsgs.APIFilterParamDesc, false)
		}
		addParam(ctx, op, "query", "sort", "", "", coremsgs.APIFilterSortDesc, false)
		addParam(ctx, op, "query", "ascending", "", "", coremsgs.APIFilterAscendingDesc, false)
		addParam(ctx, op, "query", "descending", "", "", coremsgs.APIFilterDescendingDesc, false)
		addParam(ctx, op, "query", "skip", "", "", coremsgs.APIFilterSkipDesc, false, config.GetUint(coreconfig.APIMaxFilterSkip))
		addParam(ctx, op, "query", "limit", "", config.GetString(coreconfig.APIDefaultFilterLimit), coremsgs.APIFilterLimitDesc, false, config.GetUint(coreconfig.APIMaxFilterLimit))
		addParam(ctx, op, "query", "count", "", "", coremsgs.APIFilterCountDesc, false)
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
