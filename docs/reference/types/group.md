---
layout: default
title: Group
parent: Core Resources
grand_parent: pages.reference
nav_order: 18
---

# Group
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Group

{% include_relative _includes/group_description.md %}

### Example

```json
{
    "namespace": "ns1",
    "name": "",
    "members": [
        {
            "identity": "did:firefly:org/1111",
            "node": "4f563179-b4bd-4161-86e0-c2c1c0869c4f"
        },
        {
            "identity": "did:firefly:org/2222",
            "node": "61a99af8-c1f7-48ea-8fcc-489e4822a0ed"
        }
    ],
    "localNamespace": "ns1",
    "message": "0b9dfb76-103d-443d-92fd-b114fe07c54d",
    "hash": "c52ad6c034cf5c7382d9a294f49297096a52eb55cc2da696c564b2a276633b95",
    "created": "2022-05-16T01:23:16Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `namespace` | The namespace of the group within the multiparty network | `string` |
| `name` | The optional name of the group, allowing multiple unique groups to exist with the same list of recipients | `string` |
| `members` | The list of members in this privacy group | [`Member[]`](#member) |
| `localNamespace` | The local namespace of the group | `string` |
| `message` | The message used to broadcast this group privately to the members | [`UUID`](simpletypes#uuid) |
| `hash` | The identifier hash of this group. Derived from the name and group members | `Bytes32` |
| `created` | The time when the group was first used to send a message in the network | [`FFTime`](simpletypes#fftime) |

## Member

| Field Name | Description | Type |
|------------|-------------|------|
| `identity` | The DID of the group member | `string` |
| `node` | The UUID of the node that receives a copy of the off-chain message for the identity | [`UUID`](simpletypes#uuid) |


