---
layout: default
title: WSClientActionStartPayload
parent: Core Resources
grand_parent: pages.reference
nav_order: 21
---

# WSClientActionStartPayload
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## WSClientActionStartPayload

{% include_relative _includes/wsclientactionstartpayload_description.md %}

### Example

```json
{
    "type": "start",
    "autoack": false,
    "namespace": "ns1",
    "name": "app1_subscription",
    "ephemeral": false,
    "filter": {
        "message": {},
        "transaction": {},
        "blockchainevent": {}
    },
    "options": {}
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | .type | `FFEnum`: |
| `autoack` | .autoack | `bool` |
| `namespace` | .namespace | `string` |
| `name` | .name | `string` |
| `ephemeral` | .ephemeral | `bool` |
| `filter` | .filter | [`SubscriptionFilter`](#subscriptionfilter) |
| `options` | .options | [`SubscriptionOptions`](#subscriptionoptions) |

## SubscriptionFilter

| Field Name | Description | Type |
|------------|-------------|------|
| `events` | Regular expression to apply to the event type, to subscribe to a subset of event types | `string` |
| `message` | Filters specific to message events. If an event is not a message event, these filters are ignored | [`MessageFilter`](#messagefilter) |
| `transaction` | Filters specific to events with a transaction. If an event is not associated with a transaction, this filter is ignored | [`TransactionFilter`](#transactionfilter) |
| `blockchainevent` | Filters specific to blockchain events. If an event is not a blockchain event, these filters are ignored | [`BlockchainEventFilter`](#blockchaineventfilter) |
| `topic` | Regular expression to apply to the topic of the event, to subscribe to a subset of topics. Note for messages sent with multiple topics, a separate event is emitted for each topic | `string` |
| `topics` | Deprecated: Please use 'topic' instead | `string` |
| `tag` | Deprecated: Please use 'message.tag' instead | `string` |
| `group` | Deprecated: Please use 'message.group' instead | `string` |
| `author` | Deprecated: Please use 'message.author' instead | `string` |

## MessageFilter

| Field Name | Description | Type |
|------------|-------------|------|
| `tag` | Regular expression to apply to the message 'header.tag' field | `string` |
| `group` | Regular expression to apply to the message 'header.group' field | `string` |
| `author` | Regular expression to apply to the message 'header.author' field | `string` |


## TransactionFilter

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | Regular expression to apply to the transaction 'type' field | `string` |


## BlockchainEventFilter

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | Regular expression to apply to the blockchain event 'name' field, which is the name of the event in the underlying blockchain smart contract | `string` |
| `listener` | Regular expression to apply to the blockchain event 'listener' field, which is the UUID of the event listener. So you can restrict your subscription to certain blockchain listeners. Alternatively to avoid your application need to know listener UUIDs you can set the 'topic' field of blockchain event listeners, and use a topic filter on your subscriptions | `string` |



## SubscriptionOptions

| Field Name | Description | Type |
|------------|-------------|------|
| `firstEvent` | Whether your appplication would like to receive events from the 'oldest' event emitted by your FireFly node (from the beginning of time), or the 'newest' event (from now), or a specific event sequence. Default is 'newest' | `SubOptsFirstEvent` |
| `readAhead` | The number of events to stream ahead to your application, while waiting for confirmation of consumption of those events. At least once delivery semantics are used in FireFly, so if your application crashes/reconnects this is the maximum number of events you would expect to be redelivered after it restarts | `uint16` |
| `withData` | Whether message events delivered over the subscription, should be packaged with the full data of those messages in-line as part of the event JSON payload. Or if the application should make separate REST calls to download that data. May not be supported on some transports. | `bool` |


