---
layout: default
title: Namespace
parent: Core Resources
grand_parent: pages.reference
nav_order: 20
---

# Namespace
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Namespace

{% include_relative includes/namespace_description.md %}

### Example

```json
{
    "id": "7b2d9c7e-3d60-452c-a409-05e77c855d3a",
    "name": "default",
    "description": "Default predefined namespace",
    "type": "local",
    "created": "2022-05-16T01:23:16Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the namespace. For locally established namespaces will be different on each node in the network. For broadcast namespaces, will be the same on every node | [`UUID`](simpletypes#uuid) |
| `message` | The UUID of broadcast message used to establish the namespace. Unset for local namespaces | [`UUID`](simpletypes#uuid) |
| `name` | The namespace name | `string` |
| `description` | A description of the namespace | `string` |
| `type` | The type of the namespace | `FFEnum`:<br/>`"local"`<br/>`"broadcast"`<br/>`"system"` |
| `created` | The time the namespace was created | [`FFTime`](simpletypes#fftime) |

