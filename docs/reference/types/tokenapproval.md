---
layout: default
title: TokenApproval
parent: Core Resources
grand_parent: pages.reference
nav_order: 12
---

# TokenApproval
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## TokenApproval

{% include_relative _includes/tokenapproval_description.md %}

### Example

```json
{
    "localId": "1cd3e2e2-dd6a-441d-94c5-02439de9897b",
    "pool": "1244ecbe-5862-41c3-99ec-4666a18b9dd5",
    "connector": "erc20_erc721",
    "key": "0x55860105d6a675dbe6e4d83f67b834377ba677ad",
    "operator": "0x30017fd084715e41aa6536ab777a8f3a2b11a5a1",
    "approved": true,
    "info": {
        "owner": "0x55860105d6a675dbe6e4d83f67b834377ba677ad",
        "spender": "0x30017fd084715e41aa6536ab777a8f3a2b11a5a1",
        "value": "115792089237316195423570985008687907853269984665640564039457584007913129639935"
    },
    "namespace": "ns1",
    "protocolId": "000000000032/000000/000000",
    "subject": "0x55860105d6a675dbe6e4d83f67b834377ba677ad:0x30017fd084715e41aa6536ab777a8f3a2b11a5a1",
    "active": true,
    "created": "2022-05-16T01:23:15Z",
    "tx": {
        "type": "token_approval",
        "id": "4b6e086d-0e31-482d-9683-cd18b2045031"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `localId` | The UUID of this token approval, in the local FireFly node | [`UUID`](simpletypes#uuid) |
| `pool` | The UUID the token pool this approval applies to | [`UUID`](simpletypes#uuid) |
| `connector` | The name of the token connector, as specified in the FireFly core configuration file. Required on input when there are more than one token connectors configured | `string` |
| `key` | The blockchain signing key for the approval request. On input defaults to the first signing key of the organization that operates the node | `string` |
| `operator` | The blockchain identity that is granted the approval | `string` |
| `approved` | Whether this record grants permission for an operator to perform actions on the token balance (true), or revokes permission (false) | `bool` |
| `info` | Token connector specific information about the approval operation, such as whether it applied to a limited balance of a fungible token. See your chosen token connector documentation for details | [`JSONObject`](simpletypes#jsonobject) |
| `namespace` | The namespace for the approval, which must match the namespace of the token pool | `string` |
| `protocolId` | An alphanumerically sortable string that represents this event uniquely with respect to the blockchain | `string` |
| `subject` | A string identifying the parties and entities in the scope of this approval, as provided by the token connector | `string` |
| `active` | Indicates if this approval is currently active (only one approval can be active per subject) | `bool` |
| `created` | The creation time of the token approval | [`FFTime`](simpletypes#fftime) |
| `tx` | If submitted via FireFly, this will reference the UUID of the FireFly transaction (if the token connector in use supports attaching data) | [`TransactionRef`](#transactionref) |
| `blockchainEvent` | The UUID of the blockchain event | [`UUID`](simpletypes#uuid) |
| `config` | Input only field, with token connector specific configuration of the approval.  See your chosen token connector documentation for details | [`JSONObject`](simpletypes#jsonobject) |

## TransactionRef

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | The type of the FireFly transaction | `FFEnum`: |
| `id` | The UUID of the FireFly transaction | [`UUID`](simpletypes#uuid) |


