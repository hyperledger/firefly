---
layout: default
title: TokenPool
parent: Core Resources
grand_parent: pages.reference
nav_order: 10
---

# TokenPool
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## TokenPool

{% include_relative _includes/tokenpool_description.md %}

### Example

```json
{
    "id": "90ebefdf-4230-48a5-9d07-c59751545859",
    "type": "fungible",
    "namespace": "ns1",
    "name": "my_token",
    "standard": "ERC-20",
    "locator": "address=0x056df1c53c3c00b0e13d37543f46930b42f71db0\u0026schema=ERC20WithData\u0026type=fungible",
    "decimals": 18,
    "connector": "erc20_erc721",
    "message": "43923040-b1e5-4164-aa20-47636c7177ee",
    "state": "confirmed",
    "created": "2022-05-16T01:23:15Z",
    "info": {
        "address": "0x056df1c53c3c00b0e13d37543f46930b42f71db0",
        "name": "pool8197",
        "schema": "ERC20WithData"
    },
    "tx": {
        "type": "token_pool",
        "id": "a23ffc87-81a2-4cbc-97d6-f53d320c36cd"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the token pool | [`UUID`](simpletypes#uuid) |
| `type` | The type of token the pool contains, such as fungible/non-fungible | `FFEnum`:<br/>`"fungible"`<br/>`"nonfungible"` |
| `namespace` | The namespace for the token pool | `string` |
| `name` | The name of the token pool. Note the name is not validated against the description of the token on the blockchain | `string` |
| `standard` | The ERC standard the token pool conforms to, as reported by the token connector | `string` |
| `locator` | A unique identifier for the pool, as provided by the token connector | `string` |
| `key` | The signing key used to create the token pool. On input for token connectors that support on-chain deployment of new tokens (vs. only index existing ones) this determines the signing key used to create the token on-chain | `string` |
| `symbol` | The token symbol. If supplied on input for an existing on-chain token, this must match the on-chain information | `string` |
| `decimals` | Number of decimal places that this token has | `int` |
| `connector` | The name of the token connector, as specified in the FireFly core configuration file that is responsible for the token pool. Required on input when multiple token connectors are configured | `string` |
| `message` | The UUID of the broadcast message used to inform the network to index this pool | [`UUID`](simpletypes#uuid) |
| `state` | The current state of the token pool | `FFEnum`:<br/>`"unknown"`<br/>`"pending"`<br/>`"confirmed"` |
| `created` | The creation time of the pool | [`FFTime`](simpletypes#fftime) |
| `config` | Input only field, with token connector specific configuration of the pool, such as an existing Ethereum address and block number to used to index the pool. See your chosen token connector documentation for details | [`JSONObject`](simpletypes#jsonobject) |
| `info` | Token connector specific information about the pool. See your chosen token connector documentation for details | [`JSONObject`](simpletypes#jsonobject) |
| `tx` | Reference to the FireFly transaction used to create and broadcast this pool to the network | [`TransactionRef`](#transactionref) |

## TransactionRef

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | The type of the FireFly transaction | `FFEnum`: |
| `id` | The UUID of the FireFly transaction | [`UUID`](simpletypes#uuid) |


