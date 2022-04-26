---
layout: default
title: ERC-20
parent: Use tokens
grand_parent: Tutorials
nav_order: 1
---

# Use ERC-20 tokens
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Previous steps: Start your environment
If you haven't started a FireFly stack already, please go back to the previous step and read the guide on how to [Start your environment](./setup_env.md).

[← ② Start your environment](setup_env.md){: .btn .btn-purple .mb-5}

The default Token Connector that the FireFly CLI sets up is for ERC-20 and ERC-721. If you would lik eto use one of these contracts, all you need to do is go through the first two steps in the Getting Started guide.

## Use the Sandbox (optional)
At this point you could open the Sandbox to [http://127.0.0.1:5109/home?action=tokens.pools](http://127.0.0.1:5109/home?action=tokens.pools) and perform the functions outlined in the rest of this guide. Or you can keep reading to learn how to build HTTP requests to work with tokens in FireFly.
![Tokens Sandbox](../../images/sandbox/sandbox_token_pool.png) 

## Create a pool
After you stack is up and running, the first thing you need to do is create a Token Pool. Every application will need at least one Token Pool. At a minimum, you must always
specify a `name` and `type` (`fungible` or `nonfungible`) for the pool.

### Create a new token using the sample token factory

If you are using the default ERC-20 / ERC-721 token connector, when the FireFly CLI set up your FireFly stack, it also deployed a token factory contract. This token factory contract will deploy a new ERC-20 contract when you create a Token Pool in the next step.

<div style="color: #ffffff; background: #ff7700; padding: 1em; border-radius: 5px;">⚠️ <span style="font-weight: bold;">WARNING</span>: The default token contract that was deployed by the FireFly CLI is only provided for the purpose of learning about FireFly. It is <span style="font-weight: bold;">not</span> a production grade contract. If you intend to deploy a production application using tokens FireFly you should research token contract best practices. For details, <a style="color: #ffffff;" href="https://github.com/hyperledger/firefly-tokens-erc20-erc721/blob/main/solidity/contracts/TokenFactory.sol">please see the source code</a> for the contract that was deployed.</div>

#### Request
`POST` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/pools`

```json
{
  "name": "testpool",
  "type": "fungible"
}
```

#### Response
```json
{
    "id": "5811e8d5-52d0-44b1-8b75-73f5ff88f598",
    "type": "fungible",
    "namespace": "default",
    "name": "testpool",
    "key": "0x7d41236980c3071648373126014c2f7ab587fb92",
    "connector": "erc20_erc721",
    "tx": {
        "type": "token_pool",
        "id": "1b915c2c-8d83-456b-ac2e-4b9bfbfb9dc2"
    }
}
```

Other parameters:
- You must specify a `connector` if you have configured multiple token connectors
- You may pass through a `config` object of additional parameters, if supported by your token connector
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity

To lookup the address of the new contract, you can lookup the Token Pool by its ID on the API:

#### Request
`GET` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/pools/5811e8d5-52d0-44b1-8b75-73f5ff88f598`

#### Response
```
{
    "id": "5811e8d5-52d0-44b1-8b75-73f5ff88f598",
    "type": "fungible",
    "namespace": "default",
    "name": "nft",
    "standard": "ERC20",
    "locator": "address=0xef095e65121af967f7dd7a4025debd8b2ccc0434&schema=ERC20WithData&type=fungible",
    "connector": "erc20_erc721",
    "message": "6e4a019e-625b-459e-b891-d099b9b5c622",
    "state": "confirmed",
    "created": "2022-04-22T15:38:07.914575632Z",
    "info": {
        "address": "0xef095e65121af967f7dd7a4025debd8b2ccc0434",
        "name": "testpool",
        "schema": "ERC20WithData"
    },
    "tx": {
        "type": "token_pool",
        "id": "1b915c2c-8d83-456b-ac2e-4b9bfbfb9dc2"
    }
}
```


### Creating a pool for an existing contract
If you wish to use a contract that is already on the chain, you can pass the address in a `config` object when you make the request to create the Token Pool.

`POST` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/pools`

```json
{
  "name": "testpool",
  "type": "fungible",
  "config": {
    "address": "0xb1C845D32966c79E23f733742Ed7fCe4B41901FC"
  }
}
```

## Mint tokens

Once you have a token pool, you can mint tokens within it.With a token contract deployed by the default token factory, only the creator of a pool is allowed to mint, but a different contract may define its own permission model.

`POST` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/mint`

```json
{
  "amount": 10
}
```

Other parameters:
- You must specify a `pool` name if you've created more than one pool
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity
- You may specify `to` if you'd like to send the minted tokens to a specific identity (default is the same as `key`)

## Transfer tokens

You may transfer tokens within a pool by specifying an amount and a destination understood by the connector (i.e. an Ethereum address). With a token contract deployed by the default token factory, only the owner of a token may transfer it away, but a different contract may define its own permission model.

`POST` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/transfers`

```json
{
  "amount": 1,
  "to": "0x07eab7731db665caf02bc92c286f51dea81f923f"
}
```

> **NOTE:** When transferring a non-fungible token, the amount must always be `1`. The `tokenIndex` field is also required when transferring a non-fungible token.

Other parameters:
- You must specify a `pool` name if you've created more than one pool
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity
- You may specify `from` if you'd like to send tokens from a specific identity (default is the same as `key`)

## Sending data with a transfer

All transfers (as well as mint/burn operations) support an optional `message` parameter that contains a broadcast or private
message to be sent along with the transfer. This message follows the same convention as other FireFly messages, and may be comprised
of text or blob data, and can provide context, metadata, or other supporting information about the transfer. The message will be
batched, hashed, and pinned to the primary blockchain as described in [key concepts](../keyconcepts/broadcast.html).

The message ID and hash will also be sent to the token connector as part of the transfer operation, to be written to the token blockchain
when the transaction is submitted. All recipients of the message will then be able to correlate the message with the token transfer.

`POST` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/transfers`

### Broadcast message:
```json
{
  "amount": 1,
  "to": "0x07eab7731db665caf02bc92c286f51dea81f923f",
  "message": {
    "data": [{
      "value": "payment for goods"
    }]
  }
}
```

### Private message:
```json
{
  "amount": 1,
  "to": "0x07eab7731db665caf02bc92c286f51dea81f923f",
  "message": {
    "header": {
      "type": "transfer_private",
    },
    "group": {
      "members": [{
          "identity": "org_1"
      }]
    },
    "data": [{
      "value": "payment for goods"
    }]
  }
}
```

Note that all parties in the network will be able to see the transfer (including the message ID and hash), but only
the recipients of the message will be able to view the actual message data.

## Burn tokens

You may burn tokens by simply specifying an amount. With a token contract deployed by the default token factory, only the owner of a token may
burn it, but a different contract may define its own permission model.

`POST` `http://127.0.0.1:5000/api/v1/namespaces/default/tokens/burn`

```json
{
  "amount": 1,
}
```

> **NOTE:** When burning a non-fungible token, the amount must always be `1`. The `tokenIndex` field is also required when burning a non-fungible token.

Other parameters:
- You must specify a `pool` name if you've created more than one pool
- You may specify a `key` understood by the connector (i.e. an Ethereum address) if you'd like to use a non-default signing identity
- You may specify `from` if you'd like to burn tokens from a specific identity (default is the same as `key`)
