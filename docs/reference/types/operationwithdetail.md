---
layout: default
title: OperationWithDetail
parent: Core Resources
grand_parent: pages.reference
nav_order: 8
---

# OperationWithDetail
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## OperationWithDetail

{% include_relative _includes/operationwithdetail_description.md %}

### Example

```json
{
    "id": "04a8b0c4-03c2-4935-85a1-87d17cddc20a",
    "namespace": "ns1",
    "tx": "99543134-769b-42a8-8be4-a5f8873f969d",
    "type": "sharedstorage_upload_batch",
    "status": "Succeeded",
    "plugin": "ipfs",
    "input": {
        "id": "80d89712-57f3-48fe-b085-a8cba6e0667d"
    },
    "output": {
        "payloadRef": "QmWj3tr2aTHqnRYovhS2mQAjYneRtMWJSU4M4RdAJpJwEC"
    },
    "created": "2022-05-16T01:23:15Z",
    "detail": {
        "created": "2023-01-27T17:04:24.26406392Z",
        "firstSubmit": "2023-01-27T17:04:24.419913295Z",
        "gas": "4161076",
        "gasPrice": "0",
        "history": [
            {
                "actions": [
                    {
                        "action": "AssignNonce",
                        "count": 1,
                        "lastOccurrence": "",
                        "time": ""
                    },
                    {
                        "action": "RetrieveGasPrice",
                        "count": 1,
                        "lastOccurrence": "2023-01-27T17:11:41.161213303Z",
                        "time": "2023-01-27T17:11:41.161213303Z"
                    },
                    {
                        "action": "Submit",
                        "count": 1,
                        "lastOccurrence": "2023-01-27T17:11:41.222374636Z",
                        "time": "2023-01-27T17:11:41.222374636Z"
                    }
                ],
                "subStatus": "Received",
                "time": "2023-01-27T17:11:41.122965803Z"
            },
            {
                "actions": [
                    {
                        "action": "ReceiveReceipt",
                        "count": 1,
                        "lastOccurrence": "2023-01-27T17:11:47.930332625Z",
                        "time": "2023-01-27T17:11:47.930332625Z"
                    },
                    {
                        "action": "Confirm",
                        "count": 1,
                        "lastOccurrence": "2023-01-27T17:12:02.660275549Z",
                        "time": "2023-01-27T17:12:02.660275549Z"
                    }
                ],
                "subStatus": "Tracking",
                "time": "2023-01-27T17:11:41.222400219Z"
            },
            {
                "actions": [],
                "subStatus": "Confirmed",
                "time": "2023-01-27T17:12:02.660309382Z"
            }
        ],
        "historySummary": [
            {
                "count": 1,
                "subStatus": "Received"
            },
            {
                "action": "AssignNonce",
                "count": 1
            },
            {
                "action": "RetrieveGasPrice",
                "count": 1
            },
            {
                "action": "Submit",
                "count": 1
            },
            {
                "count": 1,
                "subStatus": "Tracking"
            },
            {
                "action": "ReceiveReceipt",
                "count": 1
            },
            {
                "action": "Confirm",
                "count": 1
            },
            {
                "count": 1,
                "subStatus": "Confirmed"
            }
        ],
        "sequenceId": "0185f42f-fec8-93df-aeba-387417d477e0",
        "status": "Succeeded",
        "transactionHash": "0xfb39178fee8e725c03647b8286e6f5cb13f982abf685479a9ee59e8e9d9e51d8"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the operation | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the operation | `string` |
| `tx` | The UUID of the FireFly transaction the operation is part of | [`UUID`](simpletypes#uuid) |
| `type` | The type of the operation | `FFEnum`:<br/>`"blockchain_pin_batch"`<br/>`"blockchain_network_action"`<br/>`"blockchain_deploy"`<br/>`"blockchain_invoke"`<br/>`"sharedstorage_upload_batch"`<br/>`"sharedstorage_upload_blob"`<br/>`"sharedstorage_upload_value"`<br/>`"sharedstorage_download_batch"`<br/>`"sharedstorage_download_blob"`<br/>`"dataexchange_send_batch"`<br/>`"dataexchange_send_blob"`<br/>`"token_create_pool"`<br/>`"token_activate_pool"`<br/>`"token_transfer"`<br/>`"token_approval"` |
| `status` | The current status of the operation | `OpStatus` |
| `plugin` | The plugin responsible for performing the operation | `string` |
| `input` | The input to this operation | [`JSONObject`](simpletypes#jsonobject) |
| `output` | Any output reported back from the plugin for this operation | [`JSONObject`](simpletypes#jsonobject) |
| `error` | Any error reported back from the plugin for this operation | `string` |
| `created` | The time the operation was created | [`FFTime`](simpletypes#fftime) |
| `updated` | The last update time of the operation | [`FFTime`](simpletypes#fftime) |
| `retry` | If this operation was initiated as a retry to a previous operation, this field points to the UUID of the operation being retried | [`UUID`](simpletypes#uuid) |
| `detail` | Additional detailed information about an operation provided by the connector | `` |

