package tezos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_processArgsOk(t *testing.T) {
	methodName := "name"

	testCases := []struct {
		name             string
		processSchemaReq map[string]interface{}
		input            map[string]interface{}
	}{
		{
			name: "no input params",
			processSchemaReq: map[string]interface{}{
				"type":        "array",
				"prefixItems": []interface{}{},
			},
			input: map[string]interface{}{},
		},
		{
			name: "primitive input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varNat",
						"type": "integer",
						"details": map[string]interface{}{
							"type":         "integer",
							"internalType": "nat",
							"kind":         "option",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varNat": float64(1),
			},
		},
		{
			name: "several primitive input params",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varNat",
						"type": "integer",
						"details": map[string]interface{}{
							"type":         "integer",
							"internalType": "nat",
							"kind":         "option",
						},
					},
					map[string]interface{}{
						"name": "varString",
						"type": "string",
						"details": map[string]interface{}{
							"type":         "string",
							"internalType": "string",
							"kind":         "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varNat":    float64(1),
				"varString": "str",
			},
		},
		{
			name: "array of primitives input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varArr",
						"type": "array",
						"details": map[string]interface{}{
							"type":           "string",
							"internalType":   "string",
							"internalSchema": nil,
							"kind":           "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varArr": []interface{}{"str1", "str2", "str3"},
			},
		},
		{
			name: "struct input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varStruct",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "struct",
								"args": []interface{}{
									map[string]interface{}{
										"name": "token_id",
										"type": "nat",
									},
									map[string]interface{}{
										"name": "token_name",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varStruct": map[string]interface{}{
					"token_id":   float64(1),
					"token_name": "token name",
				},
			},
		},
		{
			name: "list of structures input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varList",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"name": "batch",
								"type": "list",
								"args": []interface{}{
									map[string]interface{}{
										"type": "struct",
										"args": []interface{}{
											map[string]interface{}{
												"name": "token_id",
												"type": "nat",
											},
											map[string]interface{}{
												"name": "token_name",
												"type": "string",
											},
										},
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varList": []interface{}{
					map[string]interface{}{
						"token_id":   float64(1),
						"token_name": "token name",
					},
					map[string]interface{}{
						"token_id":   float64(2),
						"token_name": "token name 2",
					},
				},
			},
		},
		{
			name: "variant(2 options - 1st) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"add_string": "new str",
				},
			},
		},
		{
			name: "variant(2 options - 2nd) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"remove_string": "str",
				},
			},
		},
		{
			name: "variant(3 options - 1st) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"add_string": "new str",
				},
			},
		},
		{
			name: "variant(3 options - 2nd) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"remove_string": "str",
				},
			},
		},
		{
			name: "variant(3 options - 3rd) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"edit_string": "new str",
				},
			},
		},
		{
			name: "variant(4 options - 1st) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
									"read_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"add_string": "new str",
				},
			},
		},
		{
			name: "variant(4 options - 2nd) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
									"read_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"remove_string": "str",
				},
			},
		},
		{
			name: "variant(4 options - 3rd) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
									"read_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"edit_string": "str",
				},
			},
		},
		{
			name: "variant(4 options - 4th) input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
									"read_string",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"read_string": "str",
				},
			},
		},
		{
			name: "valid map input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varMap",
						"type": "object",
						"details": map[string]interface{}{
							"type": "schema",
							"internalSchema": map[string]interface{}{
								"type": "map",
								"args": []interface{}{
									map[string]interface{}{
										"name": "key",
										"type": "integer",
									},
									map[string]interface{}{
										"name": "value",
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"varMap": map[string]interface{}{
					"mapEntries": []interface{}{
						map[string]interface{}{
							"key":   float64(1),
							"value": "val1",
						},
						map[string]interface{}{
							"key":   float64(3),
							"value": "val3",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := processArgs(tc.processSchemaReq, tc.input, methodName)
			assert.NoError(t, err)
		})
	}
}

func Test_processArgsErr(t *testing.T) {
	methodName := "name"

	testCases := []struct {
		name             string
		processSchemaReq map[string]interface{}
		input            map[string]interface{}
		expectedError    string
	}{
		{
			name:             "nil schema",
			processSchemaReq: nil,
			input:            map[string]interface{}{},
			expectedError:    "no payload schema provided",
		},
		{
			name: "nil input",
			processSchemaReq: map[string]interface{}{
				"type":        "array",
				"prefixItems": []interface{}{},
			},
			input:         nil,
			expectedError: "must specify args",
		},
		{
			name: "schema wrong root type",
			processSchemaReq: map[string]interface{}{
				"type":        "wrong",
				"prefixItems": []interface{}{},
			},
			input:         map[string]interface{}{},
			expectedError: "payload schema must define a root type of \"array\"",
		},
		{
			name: "schema nil prefixItems",
			processSchemaReq: map[string]interface{}{
				"type":        "array",
				"prefixItems": nil,
			},
			input:         map[string]interface{}{},
			expectedError: "using \"prefixItems\"",
		},
		{
			name: "empty name",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"type": "integer",
						"details": map[string]interface{}{
							"type":         "integer",
							"internalType": "nat",
						},
					},
				},
			},
			input:         map[string]interface{}{},
			expectedError: "schema must have a \"name\"",
		},
		{
			name: "wrong integer/nat type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varNat",
						"type": "integer",
						"details": map[string]interface{}{
							"type":         "integer",
							"internalType": "nat",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varNat": "wrong type",
			},
			expectedError: "invalid object passed",
		},
		{
			name: "wrong string type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varString",
						"type": "string",
						"details": map[string]interface{}{
							"type":         "string",
							"internalType": "string",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varString": struct{}{},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "wrong bytes type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varBytes",
						"type": "string",
						"details": map[string]interface{}{
							"type":         "bytes",
							"internalType": "bytes",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varBytes": struct{}{},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "wrong boolean type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varBool",
						"type": "boolean",
						"details": map[string]interface{}{
							"type":         "boolean",
							"internalType": "boolean",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varBool": struct{}{},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "wrong address type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varAddress",
						"type": "boolean",
						"details": map[string]interface{}{
							"type":         "address",
							"internalType": "address",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varAddress": struct{}{},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "wrong address type 2",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varAddress",
						"type": "boolean",
						"details": map[string]interface{}{
							"type":         "address",
							"internalType": "address",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varAddress": "wrong address",
			},
			expectedError: "unknown address type",
		},
		{
			name: "several primitive input params - wrong type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varNat",
						"type": "integer",
						"details": map[string]interface{}{
							"type":         "integer",
							"internalType": "nat",
							"kind":         "option",
						},
					},
					map[string]interface{}{
						"name": "varString",
						"type": "string",
						"details": map[string]interface{}{
							"type":         "string",
							"internalType": "string",
							"kind":         "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varNat":    float64(1),
				"varString": struct{}{},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "different array item types",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varArr",
						"type": "array",
						"details": map[string]interface{}{
							"type":         "string",
							"internalType": "string",
							"kind":         "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varArr": []interface{}{"str1", 1, struct{}{}},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "struct input param - empty schema field",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varStruct",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "struct",
								"args": []interface{}{
									map[string]interface{}{
										"name": "token_id",
										"type": "nat",
									},
									map[string]interface{}{
										"name": "token_name",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varStruct": map[string]interface{}{
					"token_name": "token name",
				},
			},
			expectedError: "wasn't found",
		},
		{
			name: "struct input param - wrong type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varStruct",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "struct",
								"args": []interface{}{
									map[string]interface{}{
										"name": "token_id",
										"type": "nat",
									},
									map[string]interface{}{
										"name": "token_name",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varStruct": map[string]interface{}{
					"token_id":   "wrong type",
					"token_name": "token name",
				},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "list of structures input param - wrong type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varList",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"name": "batch",
								"type": "list",
								"args": []interface{}{
									map[string]interface{}{
										"type": "struct",
										"args": []interface{}{
											map[string]interface{}{
												"name": "token_id",
												"type": "nat",
											},
											map[string]interface{}{
												"name": "token_name",
												"type": "string",
											},
										},
									},
								},
							},
							"kind":     "",
							"variants": nil,
						},
					},
				},
			},
			input: map[string]interface{}{
				"varList": []interface{}{
					map[string]interface{}{
						"token_id":   "wrong type",
						"token_name": "token name",
					},
					map[string]interface{}{
						"token_id":   float64(2),
						"token_name": "token name 2",
					},
				},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "invalid num of variants",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
									"read_string",
									"excess_variant",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"add_string": "str",
				},
			},
			expectedError: "wrong number of variants",
		},
		{
			name: "wrong variant type",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varVariant",
						"type": "object",
						"details": map[string]interface{}{
							"type":         "schema",
							"internalType": nil,
							"internalSchema": map[string]interface{}{
								"type": "variant",
								"variants": []interface{}{
									"add_string",
									"remove_string",
									"edit_string",
									"read_string",
									"excess_variant",
								},
								"args": []interface{}{
									map[string]interface{}{
										"name": "new_string",
										"type": "string",
									},
								},
							},
							"kind": "",
						},
					},
				},
			},
			input: map[string]interface{}{
				"varVariant": map[string]interface{}{
					"add_string": struct{}{},
				},
			},
			expectedError: "invalid object passed",
		},
		{
			name: "no mapEntries for map input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varMap",
						"type": "object",
						"details": map[string]interface{}{
							"type": "schema",
							"internalSchema": map[string]interface{}{
								"type": "map",
								"args": []interface{}{
									map[string]interface{}{
										"name": "key",
										"type": "integer",
									},
									map[string]interface{}{
										"name": "value",
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"varMap": map[string]interface{}{
					"invalid": []interface{}{
						map[string]interface{}{
							"key":   float64(1),
							"value": "val1",
						},
						map[string]interface{}{
							"key":   float64(3),
							"value": "val3",
						},
					},
				},
			},
			expectedError: "mapEntries schema property must be present",
		},
		{
			name: "value missing for map input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varMap",
						"type": "object",
						"details": map[string]interface{}{
							"type": "schema",
							"internalSchema": map[string]interface{}{
								"type": "map",
								"args": []interface{}{
									map[string]interface{}{
										"name": "key",
										"type": "integer",
									},
									map[string]interface{}{
										"name": "value",
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"varMap": map[string]interface{}{
					"mapEntries": []interface{}{
						map[string]interface{}{
							"key": float64(1),
						},
						map[string]interface{}{
							"key":   float64(3),
							"value": "val3",
						},
					},
				},
			},
			expectedError: "Schema field 'value' wasn't found",
		},
		{
			name: "key missing for map input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varMap",
						"type": "object",
						"details": map[string]interface{}{
							"type": "schema",
							"internalSchema": map[string]interface{}{
								"type": "map",
								"args": []interface{}{
									map[string]interface{}{
										"name": "key",
										"type": "integer",
									},
									map[string]interface{}{
										"name": "value",
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"varMap": map[string]interface{}{
					"mapEntries": []interface{}{
						map[string]interface{}{
							"value": "val1",
						},
						map[string]interface{}{
							"key":   float64(3),
							"value": "val3",
						},
					},
				},
			},
			expectedError: "Schema field 'key' wasn't found",
		},
		{
			name: "unknown field for map input param",
			processSchemaReq: map[string]interface{}{
				"type": "array",
				"prefixItems": []interface{}{
					map[string]interface{}{
						"name": "varMap",
						"type": "object",
						"details": map[string]interface{}{
							"type": "schema",
							"internalSchema": map[string]interface{}{
								"type": "map",
								"args": []interface{}{
									map[string]interface{}{
										"name": "key",
										"type": "integer",
									},
									map[string]interface{}{
										"name": "value",
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			input: map[string]interface{}{
				"varMap": map[string]interface{}{
					"mapEntries": []interface{}{
						map[string]interface{}{
							"key":     float64(1),
							"value":   "val1",
							"unknown": float64(1),
						},
						map[string]interface{}{
							"key":   float64(3),
							"value": "val3",
						},
					},
				},
			},
			expectedError: "Unknown schema field 'unknown' in map entry",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := processArgs(tc.processSchemaReq, tc.input, methodName)
			assert.Error(t, err)
			assert.Regexp(t, tc.expectedError, err)
		})
	}
}
