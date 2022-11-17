---
layout: default
title: Transaction
parent: Core Resources
grand_parent: pages.reference
nav_order: 6
---

# Transaction
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Transaction

{% include_relative _includes/transaction_description.md %}

### Example

```json
{
    "id": "4e7e0943-4230-4f67-89b6-181adf471edc",
    "namespace": "ns1",
    "type": "contract_invoke",
    "created": "2022-05-16T01:23:15Z",
    "blockchainIds": [
        "0x34b0327567fefed09ac7b4429549bc609302b08a9cbd8f019a078ec44447593d"
    ]
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FireFly transaction | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the FireFly transaction | `string` |
| `type` | The type of the FireFly transaction | `FFEnum`:<br/>`"none"`<br/>`"unpinned"`<br/>`"batch_pin"`<br/>`"network_action"`<br/>`"token_pool"`<br/>`"token_transfer"`<br/>`"contract_deploy"`<br/>`"contract_invoke"`<br/>`"token_approval"`<br/>`"data_publish"` |
| `created` | The time the transaction was created on this node. Note the transaction is individually created with the same UUID on each participant in the FireFly transaction | [`FFTime`](simpletypes#fftime) |
| `idempotencyKey` | An optional unique identifier for a transaction. Cannot be duplicated within a namespace, thus allowing idempotent submission of transactions to the API | `IdempotencyKey` |
| `blockchainIds` | The blockchain transaction ID, in the format specific to the blockchain involved in the transaction. Not all FireFly transactions include a blockchain. FireFly transactions are extensible to support multiple blockchain transactions | `string[]` |

