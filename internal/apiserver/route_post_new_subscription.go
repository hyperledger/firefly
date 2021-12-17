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
	"encoding/json"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/events/eifactory"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

func newSubscriptionSchemaGenerator(ctx context.Context) string {
	baseSchema, _ := openapi3gen.NewSchemaRefForValue(&fftypes.Subscription{}, nil)
	baseProps := baseSchema.Value.Properties
	delete(baseProps, "id")
	delete(baseProps, "namespace")
	delete(baseProps, "created")
	delete(baseProps, "ephemeral")
	var schemas openapi3.SchemaRefs
	for _, t := range config.GetStringSlice(config.EventTransportsEnabled) {
		transport, _ := eifactory.GetPlugin(context.Background(), t)
		if transport != nil {
			var schema openapi3.SchemaRef
			_ = json.Unmarshal([]byte(transport.GetOptionsSchema(ctx)), &schema)
			if schema.Value.Properties == nil {
				schema.Value.Properties = openapi3.Schemas{}
			}
			schema.Value.Properties["type"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    "string",
					Pattern: t,
				},
			}
			schema.Value.Properties["withData"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "boolean",
				},
			}
			var minUint16 float64
			var maxUint16 float64 = 65536
			schema.Value.Properties["readAhead"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "integer",
					Min:  &minUint16,
					Max:  &maxUint16,
				},
			}
			schema.Value.Properties["firstEvent"] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					AnyOf: openapi3.SchemaRefs{
						{
							Value: &openapi3.Schema{
								Type: "string",
								Enum: []interface{}{
									"oldest",
									"newest",
								},
							},
						},
						{Value: &openapi3.Schema{Type: "integer"}},
					},
				},
			}
			schemas = append(schemas, &schema)
		}
	}
	baseProps["options"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			OneOf: schemas,
		},
	}
	b, _ := json.Marshal(&baseSchema)
	return string(b)
}

var postNewSubscription = &oapispec.Route{
	Name:   "postNewSubscription",
	Path:   "namespaces/{ns}/subscriptions",
	Method: http.MethodPost,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.Subscription{} },
	JSONOutputValue: func() interface{} { return &fftypes.Subscription{} },
	JSONOutputCodes: []int{http.StatusCreated}, // Sync operation
	JSONInputSchema: newSubscriptionSchemaGenerator,
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		output, err = getOr(r.Ctx).CreateSubscription(r.Ctx, r.PP["ns"], r.Input.(*fftypes.Subscription))
		return output, err
	},
}
