---
layout: default
title: Blockchain Operation Status
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

Every FireFly [Transaction](./types/transaction.html) can involve zero or more [Operations](./types/operation.html). Blockchain operations are handled by the blockchain connector configured for the namespace and represent a blockchain transaction being handled by that connector.

## Blockchain Operation Status

A blockchain operation can require the connector to go through various stages of processing in order to successfully confirm the transaction on the blockchain. The orchestrator in FireFly receives updates from the connector to indicate when the operation has been completed and determine when the FireFly transaction as a whole has finished. These updates must contain enough information to correlate the operation to the FireFly transaction but it can be useful to see more detailed information about how the transaction was processed.

FireFly 1.2 introduced the concept of sub-status types that allow a blockchain connector to distinguish between the intermediate steps involved in progressing a transaction. It also introduced the concept of an action which a connector might carry out in order to progress between types of sub-status. This can be described as a state machine as shown in the following diagram:

[![Sub-status diagram](../images/blockchain-sub-status.png)](../images/blockchain-sub-status.png)

To access detailed information about a blockchain operation FireFly 1.2 introduced a new query parameter, `fetchStatus`, to the `/transaction/{txid}/operation/{opid}` API. When FireFly receives an API request that includes the fetchStatus query parameter it makes a synchronous call directly to the blockchain connector, requesting all of blockchain transaction detail it has. This payload is then included in the FireFly operation response under a new `detail` field.

### Blockchain Operation Example

```json
{
    "id": "04a8b0c4-03c2-4935-85a1-87d17cddc20a",
    "created": "2022-05-16T01:23:15Z",
    "namespace": "ns1",
    "tx": "99543134-769b-42a8-8be4-a5f8873f969d",
    "type": "blockchain_invoke",
    "status": "Succeeded",
    "plugin": "ethereum",
    "input": {
        // Input used to initiate the blockchain operation
    },
    "output": {
        // Minimal blockchain operation data necessary
        // to resolve the FF transaction
    },
    "detail": {
        // Full blockchain operation information, including sub-status
        // transitions that took place for the operation to succeed.
    }
}
```

## Detail Status Structure

The structure of a blockchain operation follows the structure described in [Operations](./types/operation.html). In FireFly 1.2, 2 new attributes were added to that structure to allow more detailed status information to be recorded:

- **history** an ordered list of status changes that have taken place during processing of the transaction
- **historySummary** an un-ordered list any sub-status type that the blockchain connector uses, and any action type that the blockchain connector carries out as part of processing the transaction.

The `history` field is designed to record an ordered list of sub-status changes that the transaction has gone through. Within each sub-status change are the actions that have been carried out to try and move the transaction on to a new sub-status. Some transactions might spend a long time going looping between different sub-status types so this field records the N most recent sub-status changes (where the size of N is determined by blockchain connector and its configuration). The follow example shows a transaction going starting at `Received`, moving to `Tracking`, and finally ending up as `Confirmed`. In order to move from `Received` to `Tracking` several actions were performed: `AssignNonce`, `RetrieveGasPrice`, and `SubmitTransaction`.

### History Example

```json
{
    ...
    "lastSubmit": "2023-01-27T17:11:41.222375469Z",
    "nonce": "14",
    "history": [
        {
            "subStatus": "Received",
            "time": "2023-01-27T17:11:41.122965803Z",
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
            "time": "2023-01-27T17:11:41.222400219Z",
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

### History Summary Example

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

## Public Chain Operations

Blockchain transactions submitted to a public chain, for example to Polygon PoS, might take longer and involve more sub-status transitions before being confirmed. One reason for this could be because of gas price fluctuations of the chain. In this case the `history` for a public blockchain operation might include a large number of `subStatus` entries. Using the example sub-status values above, a blockchain operation might move from `Tracking` to `Stale`, back to `Tracking`, back to `Stale` and so on.

Below is an example of the `history` for a public blockchain operation.

### Polygon Example

```json
{
    ...
    "lastSubmit": "2023-01-27T17:11:41.222375469Z",
    "nonce": "14",
    "history": [
        {
            "subStatus": "Received",
            "time": "2023-01-27T17:11:41.122965803Z",
            "actions": [
                {
                    "action": "AssignNonce",
                    "count": 1,
                    "lastInfo": {
                        "nonce": "1"
                    },
                    "lastOccurrence": "2023-01-27T17:11:41.122967219Z",
                    "time": "2023-01-27T17:11:41.122967136Z"
                },
                {
                    "action": "RetrieveGasPrice",
                    "count": 1,
                    "lastInfo": {
                        "gasPrice": "34422243"
                    },
                    "lastOccurrence": "2023-01-27T17:11:41.161213303Z",
                    "time": "2023-01-27T17:11:41.161213094Z"
                },
                {
                    "action": "SubmitTransaction",
                    "count": 1,
                    "lastInfo": {
                        "txHash": "0x83ba5e1cf320a1d5c949082bbec8ae7fe918e6621cec39478609ec3f7deacbdb"
                    },
                    "lastOccurrence": "2023-01-27T17:11:41.222374636Z",
                    "time": "2023-01-27T17:11:41.222374553Z"
                }
            ],
        },
        {
            "subStatus": "Tracking",
            "time": "2023-01-27T17:11:41.222400219Z",
            "actions": [],
        },
        {
            "subStatus": "Stale",
            "time": "2023-01-27T17:13:21.222100434Z",
            "actions": [
                {
                    "action": "RetrieveGasPrice",
                    "count": 1,
                    "lastInfo": {
                        "gasPrice": "44436243"
                    },
                    "lastOccurrence": "2023-01-27T17:13:22.93120838Z",
                    "time": "2023-01-27T17:13:22.93120838Z"
                },
                {
                    "action": "SubmitTransaction",
                    "count": 1,
                    "lastInfo": {
                        "txHash": "0x7b3a5e1ccbc0a1d5c949082bbec8ae7fe918e6621cec39478609ec7aea6103d5"
                    },
                    "lastOccurrence": "2023-01-27T17:13:32.656374637Z",
                    "time": "2023-01-27T17:13:32.656374637Z"
                }
            ],
        },
        {
            "subStatus": "Tracking",
            "time": "2023-01-27T17:13:33.434400219Z",
            "actions": [],
        },
        {
            "subStatus": "Stale",
            "time": "2023-01-27T17:15:21.222100434Z",
            "actions": [
                {
                    "action": "RetrieveGasPrice",
                    "count": 1,
                    "lastInfo": {
                        "gasPrice": "52129243"
                    },
                    "lastOccurrence": "2023-01-27T17:15:22.93120838Z",
                    "time": "2023-01-27T17:15:22.93120838Z"
                },
                {
                    "action": "SubmitTransaction",
                    "count": 1,
                    "lastInfo": {
                        "txHash": "0x89995e1ccbc0a1d5c949082bbec8ae7fe918e6621cec39478609ec7a8c64abc"
                    },
                    "lastOccurrence": "2023-01-27T17:15:32.656374637Z",
                    "time": "2023-01-27T17:15:32.656374637Z"
                }
            ],
        },
        {
            "subStatus": "Tracking",
            "time": "2023-01-27T17:15:33.434400219Z",
            "actions": [
                {
                    "action": "ReceiveReceipt",
                    "count": 1,
                    "lastInfo": {
                        "protocolId": "000004897621/000000"
                    },
                    "lastOccurrence": "2023-01-27T17:15:33.94120833Z",
                    "time": "2023-01-27T17:15:33.94120833Z"
                },
                {
                    "action": "Confirm",
                    "count": 1,
                    "lastOccurrence": "2023-01-27T17:16:02.780275549Z",
                    "time": "2023-01-27T17:16:02.780275382Z"
                }
            ],
        },
        {
            "subStatus": "Confirmed",
            "time": "2023-01-27T17:16:03.990309381Z",
            "actions": [],
        }
    ]
    ...
}
```