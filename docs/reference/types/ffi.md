---
layout: default
title: FFI
parent: Core Resources
grand_parent: pages.reference
nav_order: 8
---

# FFI
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## FFI

{% include_relative _includes/ffi_description.md %}

### Example

```json
{
    "id": "c35d3449-4f24-4676-8e64-91c9e46f06c4",
    "message": "e4ad2077-5714-416e-81f9-7964a6223b6f",
    "namespace": "ns1",
    "name": "SimpleStorage",
    "description": "A simple example contract in Solidity",
    "version": "v0.0.1",
    "methods": [
        {
            "id": "8f3289dd-3a19-4a9f-aab3-cb05289b013c",
            "interface": "c35d3449-4f24-4676-8e64-91c9e46f06c4",
            "name": "get",
            "namespace": "ns1",
            "pathname": "get",
            "description": "Get the current value",
            "params": [],
            "returns": [
                {
                    "name": "output",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "uint256"
                        }
                    }
                }
            ],
            "details": {
                "stateMutability": "viewable"
            }
        },
        {
            "id": "fc6f54ee-2e3c-4e56-b17c-4a1a0ae7394b",
            "interface": "c35d3449-4f24-4676-8e64-91c9e46f06c4",
            "name": "set",
            "namespace": "ns1",
            "pathname": "set",
            "description": "Set the value",
            "params": [
                {
                    "name": "newValue",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "uint256"
                        }
                    }
                }
            ],
            "returns": [],
            "details": {
                "stateMutability": "payable"
            }
        }
    ],
    "events": [
        {
            "id": "9f653f93-86f4-45bc-be75-d7f5888fbbc0",
            "interface": "c35d3449-4f24-4676-8e64-91c9e46f06c4",
            "namespace": "ns1",
            "pathname": "Changed",
            "signature": "Changed(address,uint256)",
            "name": "Changed",
            "description": "Emitted when the value changes",
            "params": [
                {
                    "name": "_from",
                    "schema": {
                        "type": "string",
                        "details": {
                            "type": "address",
                            "indexed": true
                        }
                    }
                },
                {
                    "name": "_value",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "uint256"
                        }
                    }
                }
            ]
        }
    ]
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FireFly interface (FFI) smart contract definition | [`UUID`](simpletypes#uuid) |
| `message` | The UUID of the broadcast message that was used to publish this FFI to the network | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the FFI | `string` |
| `name` | The name of the FFI - usually matching the smart contract name | `string` |
| `description` | A description of the smart contract this FFI represents | `string` |
| `version` | A version for the FFI - use of semantic versioning such as 'v1.0.1' is encouraged | `string` |
| `methods` | An array of smart contract method definitions | [`FFIMethod[]`](#ffimethod) |
| `events` | An array of smart contract event definitions | [`FFIEvent[]`](#ffievent) |

## FFIMethod

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FFI method definition | [`UUID`](simpletypes#uuid) |
| `interface` | The UUID of the FFI smart contract definition that this method is part of | [`UUID`](simpletypes#uuid) |
| `name` | The name of the method | `string` |
| `namespace` | The namespace of the FFI | `string` |
| `pathname` | The unique name allocated to this method within the FFI for use on URL paths. Supports contracts that have multiple method overrides with the same name | `string` |
| `description` | A description of the smart contract method | `string` |
| `params` | An array of method parameter/argument definitions | [`FFIParam[]`](#ffiparam) |
| `returns` | An array of method return definitions | [`FFIParam[]`](#ffiparam) |
| `details` | Additional blockchain specific fields about this method from the original smart contract. Used by the blockchain plugin and for documentation generation. | [`JSONObject`](simpletypes#jsonobject) |

## FFIParam

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract | `string` |
| `schema` | FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail | [`JSONAny`](simpletypes#jsonany) |



## FFIEvent

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FFI event definition | [`UUID`](simpletypes#uuid) |
| `interface` | The UUID of the FFI smart contract definition that this event is part of | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the FFI | `string` |
| `pathname` | The unique name allocated to this event within the FFI for use on URL paths. Supports contracts that have multiple event overrides with the same name | `string` |
| `signature` | The stringified signature of the event, as computed by the blockchain plugin | `string` |
| `name` | The name of the event | `string` |
| `description` | A description of the smart contract event | `string` |
| `params` | An array of event parameter/argument definitions | [`FFIParam[]`](#ffiparam) |
| `details` | Additional blockchain specific fields about this event from the original smart contract. Used by the blockchain plugin and for documentation generation. | [`JSONObject`](simpletypes#jsonobject) |

## FFIParam

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract | `string` |
| `schema` | FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail | [`JSONAny`](simpletypes#jsonany) |



