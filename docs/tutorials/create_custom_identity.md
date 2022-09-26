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

## Previous steps: Start your environment

If you haven't started a FireFly stack already, please go to the Getting Started guide on how to [Start your environment](../../gettingstarted/setup_env.md)

[← ② Start your environment](../../gettingstarted/setup_env.md){: .btn .btn-purple .mb-5}

## Step 1: Create a new account

The FireFly CLI has a helpful command to create an account in a local development environment for you.

> **NOTE**: In a production environment, key management actions such as creation, encryption, unlocking, etc. may be very different, depending on what type of blockchain node and signer your specific deployment is using.

To create a new account on your local stack, run:

```
ff accounts create <stack_name>
```

```
{
  "address": "0xc00109e112e21165c7065da776c75cfbc9cdc5e7",
  "privateKey": "..."
}
```

The FireFly CLI has created a new private key and address for us to be able to use, and it has loaded the encrypted private key into the signing container. However, we haven't told FireFly itself about the new key, or who it belongs to. That's what we'll do in the next steps.

## Step 2: Query the parent org for its UUID

If we want to create a new custom identity under the organizational identity that we're using in a multiparty network, first we will need to look up the UUID for our org identity. We can look that up by making a `GET` request to the status endpoint on the default namespace.

### Request

`GET` `http://localhost:5000/api/v1/status`

### Response

```js
{
    "namespace": {...},
    "node": {...},
    "org": {
        "name": "org_0",
        "registered": true,
        "did": "did:firefly:org/org_0",
        "id": "1c0abf75-0f3a-40e4-a8cd-5ff926f80aa8", // We need this in Step 3
        "verifiers": [
            {
                "type": "ethereum_address",
                "value": "0xd7320c76a2efc1909196dea876c4c7dabe49c0f4"
            }
        ]
    },
    "plugins": {...},
    "multiparty": {...}
}
```

## Step 3: Register the new custom identity with FireFly

Now we can `POST` to the identities endpoint to create a new custom identity. We will include the UUID of the organizational identity from the previous step in the `"parent"` field in the request.

### Request

`POST` `http://localhost:5000/api/v1/identities`

```js
{
    "name": "myCustomIdentity",
    "key": "0xc00109e112e21165c7065da776c75cfbc9cdc5e7", // Signing Key from Step 1
    "parent": "1c0abf75-0f3a-40e4-a8cd-5ff926f80aa8" // Org UUID from Step 2
}
```

### Response

```js
{
    "id": "5ea8f770-e004-48b5-af60-01994230ed05",
    "did": "did:firefly:myCustomIdentity",
    "type": "custom",
    "parent": "1c0abf75-0f3a-40e4-a8cd-5ff926f80aa8",
    "namespace": "",
    "name": "myCustomIdentity",
    "messages": {
        "claim": "817b7c79-a934-4936-bbb1-7dcc7c76c1f4",
        "verification": "ae55f998-49b1-4391-bed2-fa5e86dc85a2",
        "update": null
    }
}
```

## Step 4: Query the New Custom Identity

Lastly, if we want to confirm that the new identity has been created, we can query the identities endpoint to see our new custom identity.

### Request

`GET` `http://localhost:5000/api/v1/identities?fetchverifiers=true`

> **NOTE**: Using `fetchverifiers=true` will return the cryptographic verification mechanism for the FireFly identity.

### Response

```js
[
    {
        "id": "5ea8f770-e004-48b5-af60-01994230ed05",
        "did": "did:firefly:myCustomIdentity",
        "type": "custom",
        "parent": "1c0abf75-0f3a-40e4-a8cd-5ff926f80aa8",
        "namespace": "default",
        "name": "myCustomIdentity",
        "messages": {
            "claim": "817b7c79-a934-4936-bbb1-7dcc7c76c1f4",
            "verification": "ae55f998-49b1-4391-bed2-fa5e86dc85a2",
            "update": null
        },
        "created": "2022-09-19T18:10:47.365068013Z",
        "updated": "2022-09-19T18:10:47.365068013Z",
        "verifiers": [
            {
                "type": "ethereum_address",
                "value": "0xfe1ea8c8a065a0cda424e2351707c7e8eb4d2b6f"
            }
        ]
    },
    { ... },
    { ... }
]
```
