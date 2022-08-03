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

{% include_relative _includes/namespace_description.md %}

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
| `created` | The time the namespace was created | [`FFTime`](simpletypes#fftime) |

