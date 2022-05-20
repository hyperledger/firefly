---
layout: default
title: Datatype
parent: Core Resources
grand_parent: pages.reference
nav_order: 17
---

# Datatype
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Datatype

{% include_relative _includes/datatype_description.md %}

### Example

```json
{
    "id": "3a479f7e-ddda-4bda-aa24-56d06c0bf08e",
    "message": "bfcf904c-bdf7-40aa-bbd7-567f625c26c0",
    "validator": "json",
    "namespace": "ns1",
    "name": "widget",
    "version": "1.0.0",
    "hash": "639cd98c893fa45a9df6fd87bd0393a9b39e31e26fbb1eeefe90cb40c3fa02d2",
    "created": "2022-05-16T01:23:16Z",
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
        },
        "additionalProperties": false
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the datatype | [`UUID`](simpletypes#uuid) |
| `message` | The UUID of the broadcast message that was used to publish this datatype to the network | [`UUID`](simpletypes#uuid) |
| `validator` | The validator that should be used to verify this datatype | `FFEnum`:<br/>`"json"`<br/>`"none"`<br/>`"definition"` |
| `namespace` | The namespace of the datatype. Data resources can only be created referencing datatypes in the same namespace | `string` |
| `name` | The name of the datatype | `string` |
| `version` | The version of the datatype. Multiple versions can exist with the same name. Use of semantic versioning is encourages, such as v1.0.1 | `string` |
| `hash` | The hash of the value, such as the JSON schema. Allows all parties to be confident they have the exact same rules for verifying data created against a datatype | `Bytes32` |
| `created` | The time the datatype was created | [`FFTime`](simpletypes#fftime) |
| `value` | The definition of the datatype, in the syntax supported by the validator (such as a JSON Schema definition) | [`JSONAny`](simpletypes#jsonany) |

