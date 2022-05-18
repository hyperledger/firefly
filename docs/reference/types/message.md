---
layout: default
title: Message
parent: Types
grand_parent: pages.reference
nav_order: 2
---

# Message
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Message

{% include_relative includes/message_description.md %}

### Example

```json
{
    "header": {
        "id": "4ea27cce-a103-4187-b318-f7b20fd87bf3",
        "type": "broadcast",
        "namespace": "default"
    },
    "state": "confirmed",
    "data": [
        {
            "id": "fdf9f118-eb81-4086-a63d-b06715b3bb4e"
        }
    ]
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| header | The message header contains all fields that are used to build the message hash | [`MessageHeader`](#messageheader) |
| hash | The hash of the message. Derived from the header, which includes the data hash | `Bytes32` |
| batch | The UUID of the batch in which the message was pinned/transferred | `UUID` |
| state | The current state of the message | `FFEnum` |
| confirmed | The timestamp of when the message was confirmed/rejected | [`FFTime`](simpletypes#fftime) |
| data | The list of data elements attached to the message | [`DataRef[]`](dataref#dataref) |
| pins | For private messages, a unique pin hash:nonce is assigned for each topic | `string[]` |

## MessageHeader

| Field Name | Description | Type |
|------------|-------------|------|
| id | The UUID of the message. Unique to each message | `UUID` |
| cid | The correlation ID of the message. Set this when a message is a response to another message | `UUID` |
| type | The type of the message | `FFEnum` |
| txtype | The type of transaction used to order/deliver this message | `FFEnum` |
| created | The creation time of the message | [`FFTime`](simpletypes#fftime) |
| namespace | The namespace of the message | `string` |
| topics | A message topic associates this message with an ordered stream of data. A custom topic should be assigned - using the default topic is discouraged | `string[]` |
| tag | The message tag indicates the purpose of the message to the applications that process it | `string` |
| datahash | A single hash representing all data in the message. Derived from the array of data ids+hashes attached to this message | `Bytes32` |


