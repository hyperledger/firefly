---
layout: default
title: pages.blockchain_operation_status
parent: pages.reference
nav_order: 5
---

# Blockchain Operation Status
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---


## Blockchain Operations

Every FireFly [Transaction](./types/_includes/transaction_description.html) can involve zero or more [Operations](./types/_includes/operation_description.html). Blockchain operations are handled by the blockchain connector configured for the namespace and represent a blockchain transaction being handled by that connector.

## Blockchain Operation Status

A blockchain operation can require the connector to go through various stages of processing in order to successfully confirm the transaction on the blockchain. The orchestrator in FireFly receives updates from the connector to indicate when the operation has been completed and determine when the FireFly transaction as a whole has finished. These updates must contain enough information to correlate the operation to the FireFly transaction but it can be useful to see more detailed information about how the transaction was processed.

FireFly 1.2 introduced the concept of sub-status types that allow a blockchain connector to distinguish between the intermediate steps involved in progressing a transaction. It also introduced the concept of an action which a connector might carry out in order to progress between types of sub-status. This can be described as a state machine as shown in the following diagram:

[![Sub-status diagram](../images/blockchain-sub-status.png)](../images/blockchain-sub-status.png)

To access detailed information about a blockchain operation FireFly 1.2 introduced a new query parameter, `fetchStatus`, to the `/transaction/{txid}/operation/{opid}` API. When FireFly receives an API request that includes the fetchStatus query parameter it makes a synchronous call directly to the blockchain connector, requesting all of blockchain transaction detail it has. This payload is then included in the FireFly transaction response under a new `detail` field.

### Example

```json
{
    "id": "04a8b0c4-03c2-4935-85a1-87d17cddc20a",
    "namespace": "ns1",
    "tx": "99543134-769b-42a8-8be4-a5f8873f969d",
    "type": "blockchain_invoke",
    "status": "Succeeded",
    "plugin": "ethereum",
    "input": {
        "id": "80d89712-57f3-48fe-b085-a8cba6e0667d"
    },
    "output": {
        "payloadRef": "QmWj3tr2aTHqnRYovhS2mQAjYneRtMWJSU4M4RdAJpJwEC"
    },
    "detail": {
        // Blockchain operation status included here
    }
    "created": "2022-05-16T01:23:15Z"
}
```

## Status Structure

The structure of a blockchain operation follows the structure described in [Operations](./types/_includes/operation_description.html). In FireFly 1.2, 2 new attributes were added to that structure to allow more detailed status information to be recorded:

- **history** an ordered list of status changes that have taken place during processing of the transaction
- **historySummary** an un-ordered list any sub-status type that the blockchain connector uses, and any action type that the blockchain connector carries out as part of processing the transaction.

The `history` field is designed to record an ordered list of sub-status changes that the transaction has gone through. Within each sub-status change are the actions that have been carried out to try and move the transaction on to a new sub-status. Some transactions might spend a long time going looping between different sub-status types so this field records the N most recent sub-status changes (where the size of N is determined by blockchain connector and its configuration). The follow example shows a transaction going starting at `Received`, moving to `Tracking`, and finally ending up as `Confirmed`. In order to move from `Received` to `Tracking` several actions were performed: `AssignNonce`, `RetrieveGasPrice`, and `SubmitTransaction`.

### Example

```json
{
    ...
    "lastSubmit": "2023-01-27T17:11:41.222375469Z",
    "nonce": "14",
    "history": [
        {
            "subStatus": "Received",
            "time": "2023-01-27T17:11:41.122965803Z"
            "actions": [
                {
                    "action": "AssignNonce",
                    "count": 1,
                    "lastInfo": {
                        "nonce": "14"
                    },
                    "lastOccurrence": "2023-01-27T17:11:41.122967219Z",
                    "time": "2023-01-27T17:11:41.122967136Z"
                },
                {
                    "action": "RetrieveGasPrice",
                    "count": 1,
                    "lastInfo": {
                        "gasPrice": "0"
                    },
                    "lastOccurrence": "2023-01-27T17:11:41.161213303Z",
                    "time": "2023-01-27T17:11:41.161213094Z"
                },
                {
                    "action": "SubmitTransaction",
                    "count": 1,
                    "lastInfo": {
                        "txHash": "0x4c37de1cf320a1d5c949082bbec8ad5fe918e6621cec3948d609ec3f7deac243"
                    },
                    "lastOccurrence": "2023-01-27T17:11:41.222374636Z",
                    "time": "2023-01-27T17:11:41.222374553Z"
                }
            ],
        },
        {
            "subStatus": "Tracking",
            "time": "2023-01-27T17:11:41.222400219Z"
            "actions": [
                {
                    "action": "ReceiveReceipt",
                    "count": 2,
                    "lastInfo": {
                        "protocolId": "000001265122/000000"
                    },
                    "lastOccurrence": "2023-01-27T17:11:57.93120838Z",
                    "time": "2023-01-27T17:11:47.930332625Z"
                },
                {
                    "action": "Confirm",
                    "count": 1,
                    "lastOccurrence": "2023-01-27T17:12:02.660275549Z",
                    "time": "2023-01-27T17:12:02.660275382Z"
                }
            ],
        },
        {
            "subStatus": "Confirmed",
            "time": "2023-01-27T17:12:02.660309382Z",
            "actions": [],
        }
    ]
    ...
}
```

Because the `history` field is a FIFO structure describing the N most recent sub-status changes, some early sub-status changes or actions may be lost over time. For example an action of `assignNonce` might only happen once when the transaction is first processed by the connector. The `historySummary` field ensures that a minimal set of information is kept about every single subStatus type and action that has been recorded.

### Example

```json
{
    ...
    "historySummary": [
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:11:41.122966136Z",
            "lastOccurrence": "2023-01-27T17:11:41.122966136Z",
            "subStatus": "Received"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:11:41.122967219Z",
            "lastOccurrence": "2023-01-27T17:11:41.122967219Z",
            "action": "AssignNonce"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:11:41.161213303Z",
            "lastOccurrence": "2023-01-27T17:11:41.161213303Z",
            "action": "RetrieveGasPrice"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:11:41.222374636Z",
            "lastOccurrence": "2023-01-27T17:11:41.222374636Z",
            "action": "SubmitTransaction"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:11:41.222400678Z",
            "lastOccurrence": "",
            "subStatus": "Tracking"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:11:57.93120838Z",
            "lastOccurrence": "2023-01-27T17:11:57.93120838Z",
            "action": "ReceiveReceipt"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:12:02.660309382Z",
            "lastOccurrence": "2023-01-27T17:12:02.660309382Z",
            "action": "Confirm"
        },
        {
            "count": 1,
            "firstOccurrence": "2023-01-27T17:12:02.660309757Z",
            "lastOccurrence": "2023-01-27T17:12:02.660309757Z",
            "subStatus": "Confirmed"
        }
    ]
}
```
