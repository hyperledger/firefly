---
layout: default
title: WSStart
parent: Core Resources
grand_parent: pages.reference
nav_order: 21
---

# WSStart
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## WSStart

{% include_relative _includes/wsstart_description.md %}

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
| `type` | WSActionBase.type | `FFEnum`:<br/>`"start"`<br/>`"ack"`<br/>`"protocol_error"` |
| `autoack` | WSStart.autoack | `bool` |
| `namespace` | WSStart.namespace | `string` |
| `name` | WSStart.name | `string` |
| `ephemeral` | WSStart.ephemeral | `bool` |
| `filter` | WSStart.filter | [`SubscriptionFilter`](#subscriptionfilter) |
| `options` | WSStart.options | [`SubscriptionOptions`](#subscriptionoptions) |

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
| `fastack` | Webhooks only: When true the event will be acknowledged before the webhook is invoked, allowing parallel invocations | `bool` |
| `url` | Webhooks only: HTTP url to invoke. Can be relative if a base URL is set in the webhook plugin config | `string` |
| `method` | Webhooks only: HTTP method to invoke. Default=POST | `string` |
| `json` | Webhooks only: Whether to assume the response body is JSON, regardless of the returned Content-Type | `bool` |
| `reply` | Webhooks only: Whether to automatically send a reply event, using the body returned by the webhook | `bool` |
| `replytag` | Webhooks only: The tag to set on the reply message | `string` |
| `replytx` | Webhooks only: The transaction type to set on the reply message | `string` |
| `headers` | Webhooks only: Static headers to set on the webhook request | `` |
| `query` | Webhooks only: Static query params to set on the webhook request | `` |
| `input` | Webhooks only: A set of options to extract data from the first JSON input data in the incoming message. Only applies if withData=true | [`WebhookInputOptions`](#webhookinputoptions) |

## WebhookInputOptions

| Field Name | Description | Type |
|------------|-------------|------|
| `query` | A top-level property of the first data input, to use for query parameters | `string` |
| `headers` | A top-level property of the first data input, to use for headers | `string` |
| `body` | A top-level property of the first data input, to use for the request body. Default is the whole first body | `string` |
| `path` | A top-level property of the first data input, to use for a path to append with escaping to the webhook path | `string` |
| `replytx` | A top-level property of the first data input, to use to dynamically set whether to pin the response (so the requester can choose) | `string` |



