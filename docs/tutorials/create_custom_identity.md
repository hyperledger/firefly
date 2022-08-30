---
layout: default
title: Create a Custom Identity
parent: pages.tutorials
nav_order: 8
---

# Create a Custom Identity

{: .no_toc }

## Table of contents

{: .no_toc .text-delta }

1. TOC
   {:toc}

---

## Quick reference

Out of the box, a FireFly Supernode contains both an `org` and a `node` identity. Your use case might demand more granular notions of identity (ex. customers, clients, etc.). Instead of creating a Supernode for each identity, you can create multiple custom identities within a FireFly Supernode.

## Additional info

- Reference: [Identities](../reference//identities.md)
- Swagger: <a href="../swagger/swagger.html#/Default%20Namespace/postNewIdentity" data-proofer-ignore>POST /api/v1/identities</a>

### Step 1: Create a signing key for new identity

`POST` `http://localhost:5100`

`Payload`:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "personal_newAccount",
  "params": ["myCustomAccountPassword"]
}
```

`Response`:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": "0xfa140b38b81b196c8883deacd344367e17598638" // This is the new signing key
}
```

### Step 2: Unlock the new signing key on the node

> This must be repeated after every node restart

`POST` `http://localhost:5100`

`Payload`:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "personal_unlockAccount",
  "params": [
    "0xfa140b38b81b196c8883deacd344367e17598638", // From Step 1
    "myCustomAccountPassword",
    0
  ]
}
```

`Response`:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": true
}
```

### Step 3: Query the parent org for its uuid

`GET` `http://localhost:5000/api/v1/status`

`Response`:

```json
{
    "namespace": { ... },
    "node": { ... },
    "org": {
        "name": "org_0",
        "registered": true,
        "did": "did:firefly:org/org_0",
        "id": "1be22fdd-f2e3-4a33-939e-6414420cd28a", // We need this in Step 4
        "verifiers": [
            {
                "type": "ethereum_address",
                "value": "0x39768478831dac338b3363ab697290cfa4bf00ae"
            }
        ]
    },
    "plugins": { ... },
    "multiparty": { ... }
}
```

### Step 4: Register the new custom identity with FireFly

`POST` `http://localhost:5000/api/v1/identities`

`Payload`

```json
{
  "name": "myCustomIdentity",
  "key": "0xfa140b38b81b196c8883deacd344367e17598638", // Signing Key from Step 1
  "parent": "1be22fdd-f2e3-4a33-939e-6414420cd28a" // Org UUID from Step 3
}
```

`Response`:

```json
{
  "id": "333c891b-5430-4456-9ce6-c55fe66105ed",
  "did": "did:firefly:myCustomIdentity",
  "type": "custom",
  "parent": "1be22fdd-f2e3-4a33-939e-6414420cd28a",
  "namespace": "",
  "name": "myCustomIdentity",
  "messages": {
    "claim": "dde0fa34-7595-48aa-8fe3-911d56d0cd2f",
    "verification": "c563aed8-5bb3-4a46-b615-f552988f5361",
    "update": null
  }
}
```

### Step 5: Query the New Custom Identity

`GET` `http://localhost:5000/api/v1/identities`

`Response`:

```json
[
  {
    "id": "b0b9117a-26f3-4a54-88f0-9dbb4af09306",
    "did": "did:firefly:myCustomIdentity",
    "type": "custom",
    "parent": "1be22fdd-f2e3-4a33-939e-6414420cd28a",
    "namespace": "default",
    "name": "myCustomIdentity",
    "messages": {
      "claim": "da35c25d-88a4-4976-b95a-0b4a0b83eb28",
      "verification": "b8708b53-e040-4446-985d-95007ffac0e9",
      "update": null
    },
    "created": "2022-08-30T18:20:41.40493593Z",
    "updated": "2022-08-30T18:20:41.40493593Z"
  },
  { ... },
  { ... }
]
```
