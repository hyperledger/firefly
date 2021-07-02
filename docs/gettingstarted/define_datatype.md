---
layout: default
title: Define a datatype
parent: Getting Started
nav_order: 7
---

# Define a datatype
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Quick reference

As your use case matures, it is important to agree formal datatypes between
the parties. These _canonical_ datatypes need to be defined and versioned, so that
each member can extract and transform data from their internal systems into
this datatype.

Datatypes are broadcast to the network so everybody refers to the same
JSON schema when validating their data. The broadcast must complete
before a datatype can be used by an application to upload/broadcast/send data.
The same system of broadcast within FireFly is used to broadcast definitions
of datatypes, as is used to broadcast the data itself.

## Additional info

- Key Concepts: [Broadcast / shared data](/keyconcepts/broadcast.html)
- Swagger: [POST /api/v1/namespaces/{ns}/broadcast/datatype](/swagger/swagger.html#/default/postBroadcastDatatype)

### Example 1: Broadcast new datatype

`POST` `/api/v1/namespaces/{ns}/broadcast/datatype`

```json
{
  "name": "widget",
  "version": "0.0.2",
  "value": {
    "$id": "https://example.com/widget.schema.json",
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "title": "Widget",
    "type": "object",
    "properties": {
      "id": {
        "type": "string",
        "description": "The unique identifier for the widget."
      },
      "name": {
        "type": "string",
        "description": "The person's last name."
      }
    }
  }
}
```

## Example message response

Status: `202 Accepted` - a broadcast message has been sent, and on confirmation the new
datatype will be created (unless it conflicts with another definition with the same
`name` and `version` that was ordered onto the blockchain before this definition).

> _Issue [#112](https://github.com/hyperledger-labs/firefly/issues/112) proposes adding
> an option to wait for the message to be confirmed by the blockchain before returning,
> with `200 OK`._

```json
{
  "header": {
    "id": "727f7d3a-d07e-4e80-95af-59f8d2ac7531", // this is the ID of the message, not the data type
    "type": "definition", // a special type for system broadcasts
    "txtype": "batch_pin", // the broadcast is pinned to the chain
    "author": "0x0a65365587a65ce44938eab5a765fe8bc6532bdf", // the local identity
    "created": "2021-07-01T21:06:26.9997478Z", // the time the broadcast was sent
    "namespace": "ff_system", // the data/message broadcast happens on the system namespace
    "topic": [
      "ff_ns_default" // the namespace itself is used in the topic
    ],
    "tag": "ff_define_datatype", // a tag instructing FireFly to process this as a datatype definition
    "datahash": "56bd677e3e070ba62f547237edd7a90df5deaaf1a42e7d6435ec66a587c14370"
  },
  "hash": "5b6593720243831ba9e4ad002c550e95c63704b2c9dbdf31135d7d9207f8cae8",
  "local": true,
  "pending": true,
  "data": [
    {
      "id": "7539a0ab-78d8-4d42-b283-7e316b3afed3", // this data object in the ff_system namespace, contains the schema
      "hash": "22ba1cdf84f2a4aaffac665c83ff27c5431c0004dc72a9bf031ae35a75ac5aef"
    }
  ]
}
```

## Lookup the confirmed data type

`GET` `/api/v1/namespaces/default/datatypes?name=widget&version=0.0.2`

```json
[
  {
    "id": "421c94b1-66ce-4ba0-9794-7e03c63df29d", // an ID allocated to the datatype
    "message": "727f7d3a-d07e-4e80-95af-59f8d2ac7531", // the message that broadcast this data type
    "validator": "json", // the type of validator that this datatype can be used for (this one is JSON Schema)
    "namespace": "default", // the namespace of the datatype
    "name": "widget", // the name of the datatype
    "version": "0.0.2", // the version of the data type
    "hash": "a4dceb79a21937ca5ea9fa22419011ca937b4b8bc563d690cea3114af9abce2c", // hash of the schema itself
    "created": "2021-07-01T21:06:26.983986Z", // time it was confirmed
    "value": { // the JSON schema itself
      "$id": "https://example.com/widget.schema.json",
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "title": "Widget",
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "The unique identifier for the widget."
        },
        "name": {
          "type": "string",
          "description": "The person's last name."
        }
      }
    }
  }
]
```

## Example private send referring to the datatype

Once confirmed, a piece of data can be assigned that datatype and all FireFly nodes
will verify it against the schema. On a sending node, the data will be rejected at upload/send
time if it does not conform. On other nodes, bad data results in a `message_rejected` event
(rather than `message_confirmed`) for any message that arrives referring to that data.

`POST` `/api/v1/namespaces/default/send/message`

```json
{
  "header": {
    "tag": "new_widget_created",
    "topic": [ "widget_id_12345" ]
  },
  "group": {
    "members": [{
      "identity": "org_1"
    }]
  },
  "data": [
    {
      "datatype": {
        "name": "widget",
        "version": "0.0.2"
      },
      "value": {
        "id": "widget_id_12345",
        "name": "superwidget"
      }
    }
  ]
}
```
