---
layout: default
title: Operation
parent: Core Resources
grand_parent: pages.reference
nav_order: 7
---

# Operation
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Operation

{% include_relative _includes/operation_description.md %}

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
    "created": "2022-05-16T01:23:15Z"
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

