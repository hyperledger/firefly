---
layout: default
title: Identity
parent: Core Resources
grand_parent: pages.reference
nav_order: 13
---

# Identity
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---
## Identity

{% include_relative _includes/identity_description.md %}

### Example

```json
{
    "id": "114f5857-9983-46fb-b1fc-8c8f0a20846c",
    "did": "did:firefly:org/org_1",
    "type": "org",
    "parent": "688072c3-4fa0-436c-a86b-5d89673b8938",
    "namespace": "ff_system",
    "name": "org_1",
    "messages": {
        "claim": "911b364b-5863-4e49-a3f8-766dbbae7c4c",
        "verification": "24636f11-c1f9-4bbb-9874-04dd24c7502f",
        "update": null
    },
    "created": "2022-05-16T01:23:15Z"
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the identity | [`UUID`](simpletypes#uuid) |
| `did` | The DID of the identity. Unique across namespaces within a FireFly network | `string` |
| `type` | The type of the identity | `FFEnum`:<br/>`"org"`<br/>`"node"`<br/>`"custom"` |
| `parent` | The UUID of the parent identity. Unset for root organization identities | [`UUID`](simpletypes#uuid) |
| `namespace` | The namespace of the identity. Organization and node identities are always defined in the ff_system namespace | `string` |
| `name` | The name of the identity. The name must be unique within the type and namespace | `string` |
| `description` | A description of the identity. Part of the updatable profile information of an identity | `string` |
| `profile` | A set of metadata for the identity. Part of the updatable profile information of an identity | [`JSONObject`](simpletypes#jsonobject) |
| `messages` | References to the broadcast messages that established this identity and proved ownership of the associated verifiers (keys) | [`IdentityMessages`](#identitymessages) |
| `created` | The creation time of the identity | [`FFTime`](simpletypes#fftime) |
| `updated` | The last update time of the identity profile | [`FFTime`](simpletypes#fftime) |

## IdentityMessages

| Field Name | Description | Type |
|------------|-------------|------|
| `claim` | The UUID of claim message | [`UUID`](simpletypes#uuid) |
| `verification` | The UUID of claim message. Unset for root organization identities | [`UUID`](simpletypes#uuid) |
| `update` | The UUID of the most recently applied update message. Unset if no updates have been confirmed | [`UUID`](simpletypes#uuid) |


