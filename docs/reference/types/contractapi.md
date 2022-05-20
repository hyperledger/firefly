---
layout: default
title: ContractAPI
parent: Core Resources
grand_parent: pages.reference
nav_order: 7
---

# ContractAPI
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## ContractAPI

{% include_relative includes/contractapi_description.md %}

### Example

```json
{
    "interface": null,
    "name": "",
    "urls": {
        "openapi": "",
        "ui": ""
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the contract API | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the contract API | `string` |
| `interface` | Reference to the FireFly Interface definition associated with the contract API | [`FFIReference`](#ffireference) |
| `location` | If this API is tied to an individual instance of a smart contract, this field can include a blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel | [`JSONAny`](simpletypes#jsonany) |
| `name` | The name that is used in the URL to access the API | `string` |
| `message` | The UUID of the broadcast message that was used to publish this API to the network | [`UUID`](simpletypes#uuid) |
| `urls` | The URLs to use to access the API | [`ContractURLs`](#contracturls) |

## FFIReference

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FireFly interface | [`UUID`](simpletypes#uuid) |
| `name` | The name of the FireFly interface | `string` |
| `version` | The version of the FireFly interface | `string` |


## ContractURLs

| Field Name | Description | Type |
|------------|-------------|------|
| `openapi` | The URL to download the OpenAPI v3 (Swagger) description for the API generated in JSON or YAML format | `string` |
| `ui` | The URL to use in a web browser to access the SwaggerUI explorer/exerciser for the API | `string` |


