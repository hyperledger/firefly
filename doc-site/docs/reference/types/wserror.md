---
title: WSError
---
{% include-markdown "./_includes/wserror_description.md" %}

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
| `type` | WSAck.type | `FFEnum`:<br/>`"start"`<br/>`"ack"`<br/>`"protocol_error"`<br/>`"event_batch"` |
| `error` | WSAck.error | `string` |

