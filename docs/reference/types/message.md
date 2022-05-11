---
layout: default
title: Message
parent: Types
grand_parent: Reference
nav_order: 1
---

# Message
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Message

### Description

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
| header | Message.header | [`MessageHeader`](#messageheader) |
| hash | Message.hash | `Bytes32` |
| batch | Message.batch | `UUID` |
| state | Message.state | `FFEnum` |
| confirmed | Message.confirmed | [`FFTime`](#fftime) |
| data | Message.data | [`DataRef[]`](DataRef#dataref) |
| pins | Message.pins | `string[]` |

## MessageHeader

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| id | MessageHeader.id | `UUID` |
| cid | MessageHeader.cid | `UUID` |
| type | MessageHeader.type | `FFEnum` |
| txtype | MessageHeader.txtype | `FFEnum` |
| created | MessageHeader.created | [`FFTime`](#fftime) |
| namespace | MessageHeader.namespace | `string` |
| topics | MessageHeader.topics | `string[]` |
| tag | MessageHeader.tag | `string` |
| datahash | MessageHeader.datahash | `Bytes32` |

## FFTime

### Description

{% include_relative includes/fftime_description.md %}




