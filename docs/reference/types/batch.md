---
layout: default
title: Batch
parent: Core Resources
grand_parent: pages.reference
nav_order: 8
---

# Batch
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Batch

{% include_relative includes/batch_description.md %}

### Example

```json
{
    "id": "894bc0ea-0c2e-4ca4-bbca-b4c39a816bbb",
    "type": "private",
    "namespace": "ns1",
    "node": "5802ab80-fa71-4f52-9189-fb534de93756",
    "group": "cd1fedb69fb83ad5c0c62f2f5d0b04c59d2e41740916e6815a8e063b337bd32e",
    "created": "2022-05-20T02:28:41.805808Z",
    "author": "did:firefly:org/example",
    "key": "0x0a989907dcd17272257f3ebcf72f4351df65a846",
    "hash": "78d6861f860c8724468c9254b99dc09e7d9fd2d43f26f7bd40ecc9ee47be384d",
    "payload": {
        "tx": {
            "type": "private",
            "id": "04930d84-0227-4044-9d6d-82c2952a0108"
        },
        "messages": [],
        "data": []
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `hash` | The hash of the manifest of the batch | `Bytes32` |
| `payload` | Batch.payload | [`BatchPayload`](#batchpayload) |

## BatchPayload

| Field Name | Description | Type |
|------------|-------------|------|
| `tx` | BatchPayload.tx | [`TransactionRef`](#transactionref) |
| `messages` | BatchPayload.messages | [`Message[]`](message#message) |
| `data` | BatchPayload.data | [`Data[]`](data#data) |

## TransactionRef

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | The type of the FireFly transaction |  |
| `id` | The UUID of the FireFly transaction | [`UUID`](simpletypes#uuid) |



