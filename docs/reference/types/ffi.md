---
layout: default
title: FFI
parent: Types
grand_parent: Reference
nav_order: 5
---

# FFI
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## FFI

### Example

```json
{
    "name": "",
    "description": "",
    "version": ""
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| id | The UUID of the FireFly interface (FFI) smart contract definition | `UUID` |
| message | The UUID of the broadcast message that was used to publish this FFI to the network | `UUID` |
| namespace | The namespace of the FFI | `string` |
| name | The name of the FFI - usually matching the smart contract name | `string` |
| description | A description of the smart contract this FFI represents | `string` |
| version | A version for the FFI - use of semantic versioning such as 'v1.0.1' is encouraged | `string` |
| methods | An array of smart contract method definitions | [`FFIMethod[]`](#ffimethod) |
| events | An array of smart contract event definitions | [`FFIEvent[]`](#ffievent) |

## FFIMethod

| Field Name | Description | Type |
|------------|-------------|------|
| id | The UUID of the FFI method definition | `UUID` |
| interface | The UUID of the FFI smart contract definition that this method is part of | `UUID` |
| name | The name of the method | `string` |
| namespace | The namespace of the FFI | `string` |
| pathname | The unique name allocated to this method within the FFI for use on URL paths. Supports contracts that have multiple method overrides with the same name | `string` |
| description | A description of the smart contract method | `string` |
| params | An array of method parameter/argument definitions | [`FFIParam[]`](#ffiparam) |
| returns | An array of method return definitions | [`FFIParam[]`](#ffiparam) |

## FFIParam

| Field Name | Description | Type |
|------------|-------------|------|
| name | The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract | `string` |
| schema | FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail | `JSONAny` |



## FFIEvent

| Field Name | Description | Type |
|------------|-------------|------|
| id | The UUID of the FFI event definition | `UUID` |
| interface | The UUID of the FFI smart contract definition that this event is part of | `UUID` |
| namespace | The namespace of the FFI | `string` |
| pathname | The unique name allocated to this event within the FFI for use on URL paths. Supports contracts that have multiple event overrides with the same name | `string` |
| signature | The stringified signature of the event, as computed by the blockchain plugin | `string` |


