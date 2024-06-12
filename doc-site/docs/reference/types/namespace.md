---
title: Namespace
---
{% include-markdown "./_includes/namespace_description.md" %}

### Example

```json
{
    "name": "default",
    "networkName": "default",
    "description": "Default predefined namespace",
    "created": "2022-05-16T01:23:16Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The local namespace name | `string` |
| `networkName` | The shared namespace name within the multiparty network | `string` |
| `description` | A description of the namespace | `string` |
| `created` | The time the namespace was created | [`FFTime`](simpletypes.md#fftime) |

