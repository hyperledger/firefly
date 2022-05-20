---
layout: default
title: WSError
parent: Core Resources
grand_parent: pages.reference
nav_order: 23
---

# WSError
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## WSError

{% include_relative _includes/wserror_description.md %}

### Example

```json
{
    "type": "protocol_error",
    "error": "FF10175: Acknowledgment does not match an inflight event + subscription"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `type` | WSAck.type | `FFEnum`:<br/>`"start"`<br/>`"ack"`<br/>`"protocol_error"` |
| `error` | WSAck.error | `string` |

