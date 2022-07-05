---
layout: default
title: TokenTransfer
parent: Core Resources
grand_parent: pages.reference
nav_order: 11
---

# TokenTransfer
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## TokenTransfer

{% include_relative _includes/tokentransfer_description.md %}

### Example

```json
{
    "type": "transfer",
    "pool": "1244ecbe-5862-41c3-99ec-4666a18b9dd5",
    "uri": "firefly://token/1",
    "connector": "erc20_erc721",
    "namespace": "ns1",
    "key": "0x55860105D6A675dBE6e4d83F67b834377Ba677AD",
    "from": "0x55860105D6A675dBE6e4d83F67b834377Ba677AD",
    "to": "0x55860105D6A675dBE6e4d83F67b834377Ba677AD",
    "amount": "1000000000000000000",
    "protocolId": "000000000041/000000/000000",
    "message": "780b9b90-e3b0-4510-afac-b4b1f2940b36",
    "messageHash": "780204e634364c42779920eddc8d9fecccb33e3607eeac9f53abd1b31184ae4e",
    "created": "2022-05-16T01:23:15Z",
    "tx": {
        "type": "token_transfer",
        "id": "62767ca8-99f9-439c-9deb-d80c6672c158"
    },
    "blockchainEvent": "b57fcaa2-156e-4c3f-9b0b-ddec9ee25933"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | The type of transfer such as mint/burn/transfer | `FFEnum`:<br/>`"mint"`<br/>`"burn"`<br/>`"transfer"` |
| `localId` | The UUID of this token transfer, in the local FireFly node | [`UUID`](simpletypes#uuid) |
| `pool` | The UUID the token pool this transfer applies to | [`UUID`](simpletypes#uuid) |
| `tokenIndex` | The index of the token within the pool that this transfer applies to | `string` |
| `uri` | The URI of the token this transfer applies to | `string` |
| `connector` | The name of the token connector, as specified in the FireFly core configuration file. Required on input when there are more than one token connectors configured | `string` |
| `namespace` | The namespace for the transfer, which must match the namespace of the token pool | `string` |
| `key` | The blockchain signing key for the transfer. On input defaults to the first signing key of the organization that operates the node | `string` |
| `from` | The source account for the transfer. On input defaults to the value of 'key' | `string` |
| `to` | The target account for the transfer. On input defaults to the value of 'key' | `string` |
| `amount` | The amount for the transfer. For non-fungible tokens will always be 1. For fungible tokens, the number of decimals for the token pool should be considered when inputting the amount. For example, with 18 decimals a fractional balance of 10.234 will be specified as 10,234,000,000,000,000,000 | [`FFBigInt`](simpletypes#ffbigint) |
| `protocolId` | An alphanumerically sortable string that represents this event uniquely with respect to the blockchain | `string` |
| `message` | The UUID of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector | [`UUID`](simpletypes#uuid) |
| `messageHash` | The hash of a message that has been correlated with this transfer using the data field of the transfer in a compatible token connector | `Bytes32` |
| `created` | The creation time of the transfer | [`FFTime`](simpletypes#fftime) |
| `tx` | If submitted via FireFly, this will reference the UUID of the FireFly transaction (if the token connector in use supports attaching data) | [`TransactionRef`](#transactionref) |
| `blockchainEvent` | The UUID of the blockchain event | [`UUID`](simpletypes#uuid) |
| `config` | Input only field, with token connector specific configuration of the transfer. See your chosen token connector documentation for details | [`JSONObject`](simpletypes#jsonobject) |

## TransactionRef

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | The type of the FireFly transaction | `FFEnum`: |
| `id` | The UUID of the FireFly transaction | [`UUID`](simpletypes#uuid) |


