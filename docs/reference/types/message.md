---
layout: default
title: Message
parent: Core Resources
grand_parent: pages.reference
nav_order: 15
---

# Message
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Message

{% include_relative _includes/message_description.md %}

### Example

```json
{
    "header": {
        "id": "4ea27cce-a103-4187-b318-f7b20fd87bf3",
        "cid": "00d20cba-76ed-431d-b9ff-f04b4cbee55c",
        "type": "private",
        "txtype": "batch_pin",
        "author": "did:firefly:org/acme",
        "key": "0xD53B0294B6a596D404809b1d51D1b4B3d1aD4945",
        "created": "2022-05-16T01:23:10Z",
        "namespace": "ns1",
        "group": "781caa6738a604344ae86ee336ada1b48a404a85e7041cf75b864e50e3b14a22",
        "topics": [
            "topic1"
        ],
        "tag": "blue_message",
        "datahash": "c07be180b147049baced0b6219d9ce7a84ab48f2ca7ca7ae949abb3fe6491b54"
    },
    "localNamespace": "ns1",
    "state": "confirmed",
    "confirmed": "2022-05-16T01:23:16Z",
    "data": [
        {
            "id": "fdf9f118-eb81-4086-a63d-b06715b3bb4e",
            "hash": "34cf848d896c83cdf433ea7bd9490c71800b316a96aac3c3a78a42a4c455d67d"
        }
    ]
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `header` | The message header contains all fields that are used to build the message hash | [`MessageHeader`](#messageheader) |
| `localNamespace` | The local namespace of the message | `string` |
| `hash` | The hash of the message. Derived from the header, which includes the data hash | `Bytes32` |
| `batch` | The UUID of the batch in which the message was pinned/transferred | [`UUID`](simpletypes#uuid) |
| `state` | The current state of the message | `FFEnum`:<br/>`"staged"`<br/>`"ready"`<br/>`"sent"`<br/>`"pending"`<br/>`"confirmed"`<br/>`"rejected"` |
| `confirmed` | The timestamp of when the message was confirmed/rejected | [`FFTime`](simpletypes#fftime) |
| `data` | The list of data elements attached to the message | [`DataRef[]`](#dataref) |
| `pins` | For private messages, a unique pin hash:nonce is assigned for each topic | `string[]` |
| `idempotencyKey` | An optional unique identifier for a message. Cannot be duplicated within a namespace, thus allowing idempotent submission of messages to the API. Local only - not transferred when the message is sent to other members of the network | `IdempotencyKey` |

## MessageHeader

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the message. Unique to each message | [`UUID`](simpletypes#uuid) |
| `cid` | The correlation ID of the message. Set this when a message is a response to another message | [`UUID`](simpletypes#uuid) |
| `type` | The type of the message | `FFEnum`:<br/>`"definition"`<br/>`"broadcast"`<br/>`"private"`<br/>`"groupinit"`<br/>`"transfer_broadcast"`<br/>`"transfer_private"` |
| `txtype` | The type of transaction used to order/deliver this message | `FFEnum`:<br/>`"none"`<br/>`"unpinned"`<br/>`"batch_pin"`<br/>`"network_action"`<br/>`"token_pool"`<br/>`"token_transfer"`<br/>`"contract_deploy"`<br/>`"contract_invoke"`<br/>`"token_approval"`<br/>`"data_publish"` |
| `author` | The DID of identity of the submitter | `string` |
| `key` | The on-chain signing key used to sign the transaction | `string` |
| `created` | The creation time of the message | [`FFTime`](simpletypes#fftime) |
| `namespace` | The namespace of the message within the multiparty network | `string` |
| `topics` | A message topic associates this message with an ordered stream of data. A custom topic should be assigned - using the default topic is discouraged | `string[]` |
| `tag` | The message tag indicates the purpose of the message to the applications that process it | `string` |
| `datahash` | A single hash representing all data in the message. Derived from the array of data ids+hashes attached to this message | `Bytes32` |


## DataRef

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the referenced data resource | [`UUID`](simpletypes#uuid) |
| `hash` | The hash of the referenced data | `Bytes32` |


