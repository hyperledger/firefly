---
layout: default
title: WSClientActionAckPayload
parent: Core Resources
grand_parent: pages.reference
nav_order: 22
---

# WSClientActionAckPayload
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## WSClientActionAckPayload

{% include_relative _includes/wsclientactionackpayload_description.md %}

### Example

```json
{
    "type": "ack",
    "id": "f78bf82b-1292-4c86-8a08-e53d855f1a64",
    "subscription": {
        "id": null,
        "namespace": "ns1",
        "name": "app1_subscription"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | .type | `FFEnum`: |
| `id` | .id | [`UUID`](simpletypes#uuid) |
| `subscription` | .subscription | [`SubscriptionRef`](#subscriptionref) |

## SubscriptionRef

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the subscription | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the subscription. A subscription will only receive events generated in the namespace of the subscription | `string` |
| `name` | The name of the subscription. The application specifies this name when it connects, in order to attach to the subscription and receive events that arrived while it was disconnected. If multiple apps connect to the same subscription, events are workload balanced across the connected application instances | `string` |


