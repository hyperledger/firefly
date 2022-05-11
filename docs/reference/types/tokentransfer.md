---
layout: default
title: TokenTransfer
parent: Types
grand_parent: Reference
nav_order: 2
---

# TokenTransfer
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## TokenTransfer

### Example
```json
{
    "type": "transfer",
    "pool": "1244ecbe-5862-41c3-99ec-4666a18b9dd5",
    "from": "0x98151D8AB3af082A5DC07746C220Fb6C95Bc4a50",
    "to": "0x7b746b92869De61649d148823808653430682C0d",
    "amount": "0",
    "message": "855af8e7-2b02-4e05-ad7d-9ae0d4c409ba",
    "tx": {
        "type": ""
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| type | TokenTransfer.type | `FFEnum` |
| localId | TokenTransfer.localId | `UUID` |
| pool | TokenTransfer.pool | `UUID` |
| tokenIndex | TokenTransfer.tokenIndex | `string` |
| uri | TokenTransfer.uri | `string` |
| connector | TokenTransfer.connector | `string` |
| namespace | TokenTransfer.namespace | `string` |
| key | TokenTransfer.key | `string` |
| from | TokenTransfer.from | `string` |
| to | TokenTransfer.to | `string` |
| amount | TokenTransfer.amount | [`FFBigInt`](#ffbigint) |
| protocolId | TokenTransfer.protocolId | `string` |
| message | TokenTransfer.message | `UUID` |
| messageHash | TokenTransfer.messageHash | `Bytes32` |
| created | TokenTransfer.created | [`FFTime`](#fftime) |
| tx | TokenTransfer.tx | [`TransactionRef`](#transactionref) |
| blockchainEvent | TokenTransfer.blockchainEvent | `UUID` |

## FFBigInt



## FFTime

### Description

{% include_relative includes/fftime_description.md %}



## TransactionRef

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| type | Transaction.type | `FFEnum` |
| id | Transaction.id | `UUID` |


