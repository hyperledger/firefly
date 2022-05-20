---
layout: default
title: Verifier
parent: Core Resources
grand_parent: pages.reference
nav_order: 14
---

# Verifier
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Verifier

{% include_relative _includes/verifier_description.md %}

### Example

```json
{
    "hash": "6818c41093590b862b781082d4df5d4abda6d2a4b71d737779edf6d2375d810b",
    "identity": "114f5857-9983-46fb-b1fc-8c8f0a20846c",
    "type": "ethereum_address",
    "value": "0x30017fd084715e41aa6536ab777a8f3a2b11a5a1",
    "created": "2022-05-16T01:23:15Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `hash` | Hash used as a globally consistent identifier for this namespace + type + value combination on every node in the network | `Bytes32` |
| `identity` | The UUID of the parent identity that has claimed this verifier | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the verifier | `string` |
| `type` | The type of the verifier | `FFEnum`:<br/>`"ethereum_address"`<br/>`"fabric_msp_id"`<br/>`"dx_peer_id"` |
| `value` | The verifier string, such as an Ethereum address, or Fabric MSP identifier | `string` |
| `created` | The time this verifier was created on this node | [`FFTime`](simpletypes#fftime) |

