---
layout: default
title: ContractListener
parent: Core Resources
grand_parent: pages.reference
nav_order: 9
---

# ContractListener
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## ContractListener

{% include_relative _includes/contractlistener_description.md %}

### Example

```json
{
    "id": "d61980a9-748c-4c72-baf5-8b485b514d59",
    "interface": {
        "id": "ff1da3c1-f9e7-40c2-8d93-abb8855e8a1d"
    },
    "namespace": "ns1",
    "name": "contract1_events",
    "backendId": "sb-dd8795fc-a004-4554-669d-c0cf1ee2c279",
    "location": {
        "address": "0x596003a91a97757ef1916c8d6c0d42592630d2cf"
    },
    "created": "2022-05-16T01:23:15Z",
    "event": {
        "name": "Changed",
        "description": "",
        "params": [
            {
                "name": "x",
                "schema": {
                    "type": "integer",
                    "details": {
                        "type": "uint256",
                        "internalType": "uint256"
                    }
                }
            }
        ]
    },
    "signature": "Changed(uint256)",
    "topic": "app1_topic",
    "options": {
        "firstEvent": "newest"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the smart contract listener | [`UUID`](simpletypes#uuid) |
| `interface` | A reference to an existing FFI, containing pre-registered type information for the event | [`FFIReference`](#ffireference) |
| `namespace` | The namespace of the listener, which defines the namespace of all blockchain events detected by this listener | `string` |
| `name` | A descriptive name for the listener | `string` |
| `backendId` | An ID assigned by the blockchain connector to this listener | `string` |
| `location` | A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel | [`JSONAny`](simpletypes#jsonany) |
| `created` | The creation time of the listener | [`FFTime`](simpletypes#fftime) |
| `event` | The definition of the event, either provided in-line when creating the listener, or extracted from the referenced FFI | [`FFISerializedEvent`](#ffiserializedevent) |
| `signature` | The stringified signature of the event, as computed by the blockchain plugin | `string` |
| `topic` | A topic to set on the FireFly event that is emitted each time a blockchain event is detected from the blockchain. Setting this topic on a number of listeners allows applications to easily subscribe to all events they need | `string` |
| `options` | Options that control how the listener subscribes to events from the underlying blockchain | [`ContractListenerOptions`](#contractlisteneroptions) |

## FFIReference

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FireFly interface | [`UUID`](simpletypes#uuid) |
| `name` | The name of the FireFly interface | `string` |
| `version` | The version of the FireFly interface | `string` |


## FFISerializedEvent

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the event | `string` |
| `description` | A description of the smart contract event | `string` |
| `params` | An array of event parameter/argument definitions | [`FFIParam[]`](#ffiparam) |
| `details` | Additional blockchain specific fields about this event from the original smart contract. Used by the blockchain plugin and for documentation generation. | [`JSONObject`](simpletypes#jsonobject) |

## FFIParam

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract | `string` |
| `schema` | FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail | [`JSONAny`](simpletypes#jsonany) |



## ContractListenerOptions

| Field Name | Description | Type |
|------------|-------------|------|
| `firstEvent` | A blockchain specific string, such as a block number, to start listening from. The special strings 'oldest' and 'newest' are supported by all blockchain connectors. Default is 'newest' | `string` |


