---
layout: default
title: Event
parent: Core Resources
grand_parent: pages.reference
nav_order: 2
---

# Event
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Event

{% include_relative _includes/event_description.md %}

### Example

```json
{
    "id": "5f875824-b36b-4559-9791-a57a2e2b30dd",
    "sequence": 168,
    "type": "transaction_submitted",
    "namespace": "ns1",
    "reference": "0d12aa75-5ed8-48a7-8b54-45274c6edcb1",
    "tx": "0d12aa75-5ed8-48a7-8b54-45274c6edcb1",
    "topic": "batch_pin",
    "created": "2022-05-16T01:23:15Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID assigned to this event by your local FireFly node | [`UUID`](simpletypes#uuid) |
| `sequence` | A sequence indicating the order in which events are delivered to your application. Assure to be unique per event in your local FireFly database (unlike the created timestamp) | `int64` |
| `type` | All interesting activity in FireFly is emitted as a FireFly event, of a given type. The 'type' combined with the 'reference' can be used to determine how to process the event within your application | `FFEnum`:<br/>`"transaction_submitted"`<br/>`"message_confirmed"`<br/>`"message_rejected"`<br/>`"datatype_confirmed"`<br/>`"identity_confirmed"`<br/>`"identity_updated"`<br/>`"token_pool_confirmed"`<br/>`"token_pool_op_failed"`<br/>`"token_transfer_confirmed"`<br/>`"token_transfer_op_failed"`<br/>`"token_approval_confirmed"`<br/>`"token_approval_op_failed"`<br/>`"contract_interface_confirmed"`<br/>`"contract_api_confirmed"`<br/>`"blockchain_event_received"`<br/>`"blockchain_invoke_op_succeeded"`<br/>`"blockchain_invoke_op_failed"`<br/>`"blockchain_contract_deploy_op_succeeded"`<br/>`"blockchain_contract_deploy_op_failed"` |
| `namespace` | The namespace of the event. Your application must subscribe to events within a namespace | `string` |
| `reference` | The UUID of an resource that is the subject of this event. The event type determines what type of resource is referenced, and whether this field might be unset | [`UUID`](simpletypes#uuid) |
| `correlator` | For message events, this is the 'header.cid' field from the referenced message. For certain other event types, a secondary object is referenced such as a token pool | [`UUID`](simpletypes#uuid) |
| `tx` | The UUID of a transaction that is event is part of. Not all events are part of a transaction | [`UUID`](simpletypes#uuid) |
| `topic` | A stream of information this event relates to. For message confirmation events, a separate event is emitted for each topic in the message. For blockchain events, the listener specifies the topic. Rules exist for how the topic is set for other event types | `string` |
| `created` | The time the event was emitted. Not guaranteed to be unique, or to increase between events in the same order as the final sequence events are delivered to your application. As such, the 'sequence' field should be used instead of the 'created' field for querying events in the exact order they are delivered to applications | [`FFTime`](simpletypes#fftime) |

