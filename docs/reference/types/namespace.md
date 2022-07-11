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
    "description": "Default predefined namespace",
    "created": "2022-05-16T01:23:16Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The namespace name | `string` |
| `description` | A description of the namespace | `string` |
| `created` | The time the namespace was created | [`FFTime`](simpletypes#fftime) |

