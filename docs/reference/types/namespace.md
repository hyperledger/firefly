---
layout: default
title: Namespace
parent: Core Resources
grand_parent: pages.reference
nav_order: 20
---

# Namespace
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Namespace

{% include_relative _includes/namespace_description.md %}

### Example

```json
{
    "id": "7b2d9c7e-3d60-452c-a409-05e77c855d3a",
    "name": "default",
    "description": "Default predefined namespace",
    "type": "local",
    "created": "2022-05-16T01:23:16Z",
    "fireflyContract": {
        "active": {
            "index": 0
        }
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the namespace. For locally established namespaces will be different on each node in the network. For broadcast namespaces, will be the same on every node | [`UUID`](simpletypes#uuid) |
| `message` | The UUID of broadcast message used to establish the namespace. Unset for local namespaces | [`UUID`](simpletypes#uuid) |
| `name` | The namespace name | `string` |
| `description` | A description of the namespace | `string` |
| `type` | The type of the namespace | `FFEnum`:<br/>`"local"`<br/>`"broadcast"`<br/>`"system"` |
| `created` | The time the namespace was created | [`FFTime`](simpletypes#fftime) |
| `fireflyContract` | Info on the FireFly smart contract configured for this namespace | [`FireFlyContracts`](#fireflycontracts) |

## FireFlyContracts

| Field Name | Description | Type |
|------------|-------------|------|
| `active` | The currently active FireFly smart contract | [`FireFlyContractInfo`](#fireflycontractinfo) |
| `terminated` | Previously-terminated FireFly smart contracts | [`FireFlyContractInfo[]`](#fireflycontractinfo) |

## FireFlyContractInfo

| Field Name | Description | Type |
|------------|-------------|------|
| `index` | The index of this contract in the config file | `int` |
| `finalEvent` | The identifier for the final blockchain event received from this contract before termination | `string` |
| `location` | A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel | [`JSONAny`](simpletypes#jsonany) |
| `firstEvent` | A blockchain specific string, such as a block number, to start listening from. The special strings 'oldest' and 'newest' are supported by all blockchain connectors | `string` |
| `subscription` | The UUID of the subscription for the FireFly BatchPin contract | `string` |



