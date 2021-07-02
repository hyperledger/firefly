---
layout: default
title: Explore messages
parent: Getting Started
nav_order: 5
---

# Explore messages
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Quick reference

The FireFly Explorer is a great way to view the messages sent and received by your node.

Just open `/ui` on your FireFly node to access it.

![Explore Messages](../images/message_view.png)

This builds on the APIs to query and filter messages, described below

## Additional info

- Reference: [API Query Syntax](/reference/api_query_syntax.html)
- Swagger: [GET /api/v1/namespaces/{ns}/messages](/swagger/swagger.html#/default/getMsgs)

### Example 1: Query confirmed messages

These are the messages ready to be processed in your application.
All data associated with the message (including BLOB attachments) is available,
and if they are sequenced by the blockchain, then those blockchain transactions
are complete.

> _The order in which you process messages should be determined by absolute
> order of `message_confirmed` events - queryable via the `events` collection, or
> through event listeners (discussed next in the getting started guide)._
> 
> _That is because `messages` are ordered by timestamp,
> which is potentially subject to adjustments of the clock.
> Whereas `events` are ordered by the insertion order into the database, and as such
> changes in the clock do not affect the order._

`GET` `/api/v1/namespaces/{ns}/messages?pending=false&limit=100`

## Example response

```json
[
  {
    "header": {
      "id": "423302bb-abfc-4d64-892d-38b2fdfe1549",
      "type": "private", // this was a private send
      "txtype": "batch_pin", // pinned in a batch to the blockchain
      "author": "0x1d14b65d2dd5c13f6cb6d3dc4aa13c795a8f3b28",
      "created": "2021-07-02T03:09:40.2606238Z",
      "namespace": "default",
      "group": "2aa5297b5eed0c3a612a667c727ca38b54fb3b5cc245ebac4c2c7abe490bdf6c", // sent to this group
      "topic": [
        "widget_id_12345"
      ],
      "tag": "new_widget_created",
      "datahash": "551dd261e80ce76b1908c031cff8a707bd76376d6eddfdc1040c2ed6481ec8dd"
    },
    "hash": "bf2ca94db8c31bae3cae974bb626fa822c6eee5f572d274d72281e72537b30b3",
    "batch": "f7ac773d-885a-4d73-ac6b-c09f5346a051", // the batch ID that pinned this message to the chain
    "pending": false, // message is now confirmed
    "confirmed": "2021-07-02T03:09:49.9207211Z", // timestamp when this node confirmed the message
    "data": [
      {
        "id": "914eed77-8789-451c-b55f-ba9570a71eba",
        "hash": "9541cabc750c692e553a421a6c5c07ebcae820774d2d8d0b88fac2a231c10bf2"
      }
    ],
    "pins": [
      // A "pin" is an identifier that is used by FireFly for sequencing messages.
      //
      // For private messages, it is an obfuscated representation of the sequence of this message,
      // on a topic, within this group, from this sender. There will be one pin per topic. You will find these
      // pins in the blockchain transaction, as well as the off-chain data.
      // Each one is unqiue, and without the group hash, very difficult to correlate - meaning
      // the data on-chain provides a high level of privacy.
      //
      // Note for broadcast (which does not require obfuscation), it is simply a hash of the topic.
      // So you will see the same pin for all messages on the same topic.
      "ee56de6241522ab0ad8266faebf2c0f1dc11be7bd0c41d847998135b45685b77"
    ]
  }
]
```

### Example 2: Query all messages

The natural sort order the API will return for messages is:
- Pending messages first
  - In descending `created` timestamp order
- Confirmed messages
  - In descending `confirmed` timestamp order

`GET` `/api/v1/namespaces/{ns}/messages`

