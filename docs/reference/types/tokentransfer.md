---
layout: default
title: TokenTransfer
parent: Types
grand_parent: pages.reference
nav_order: 3
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
| type | The type of transfer such as mint/burn/transfer | `FFEnum` |
| localId | The UUID of this token transfer, in the local FireFly node | `UUID` |
| pool | The UUID the token pool this transfer applies to | `UUID` |
| tokenIndex | The index of the token within the pool that this transfer applies to | `string` |
| uri | The URI of the token this transfer applies to | `string` |
| connector | The name of the token connector, as specified in the FireFly core configuration file. Required on input when there are more than one token connectors configured | `string` |
| namespace | The namespace for the transfer, which must match the namespace of the token pool | `string` |
| key | The blockchain signing key for the transfer. On input defaults to the first signing key of the organization that operates the node | `string` |
| from | The source account for the transfer. On input defaults to the value of 'key' | `string` |
| to | The target account for the transfer. On input defaults to the value of 'key' | `string` |
| amount | The amount for the transfer. For non-fungible tokens will always be 1. For fungible tokens, the number of decimals for the token pool should be considered when inputting the amount. For example, with 18 decimals a fractional balance of 10.234 will be specified as 10,234,000,000,000,000,000 | [`FFBigInt`](simpletypes#ffbigint) |
| protocolId | An alphanumerically sortable string that represents this event uniquely with respect to the blockchain | `string` |
| message | The UUID of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector | `UUID` |
| messageHash | The hash of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector | `Bytes32` |
| created | The creation time of the transfer | [`FFTime`](simpletypes#fftime) |
| tx | If submitted via FireFly, this will reference the UUID of the FireFly transaction (if the token connector in use supports attaching data) | [`TransactionRef`](#transactionref) |
| blockchainEvent | The UUID of the blockchain event | `UUID` |

## TransactionRef

| Field Name | Description | Type |
|------------|-------------|------|
| type | The type of the FireFly transaction | `FFEnum` |
| id | The UUID of the FireFly transaction | `UUID` |


