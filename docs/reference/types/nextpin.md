---
layout: default
title: NextPin
parent: Core Resources
grand_parent: pages.reference
nav_order: 20
---

# NextPin
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## NextPin

{% include_relative _includes/nextpin_description.md %}

### Example

```json
{
    "namespace": "ns1",
    "context": "a25b65cfe49e5ed78c256e85cf07c96da938144f12fcb02fe4b5243a4631bd5e",
    "identity": "did:firefly:org/example",
    "hash": "00e55c63905a59782d5bc466093ead980afc4a2825eb68445bcf1312cc3d6de2",
    "nonce": 12345
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `namespace` | The namespace of the next-pin | `string` |
| `context` | The context the next-pin applies to - the hash of the privacy group-hash + topic. The group-hash is only known to the participants (can itself contain a salt in the group-name). This context is combined with the member and nonce to determine the final hash that is written on-chain | `Bytes32` |
| `identity` | The member of the privacy group the next-pin applies to | `string` |
| `hash` | The unique masked pin string | `Bytes32` |
| `nonce` | The numeric index - which is monotonically increasing for each member of the privacy group | `int64` |

