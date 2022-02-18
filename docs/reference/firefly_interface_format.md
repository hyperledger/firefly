---
layout: default
title: FireFly Interface Format
parent: Reference
nav_order: 3
---

# FireFly Interface Format
{: .no_toc }

FireFly defines a common, blockchain agnostic way to describe smart contracts. This is referred to as a **Contract Interface**, and it is written in the FireFly Interface (FFI) format. It is a simple JSON document that has a name, a namespace, a version, a list of methods, and a list of events.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

There are four required fields when broadcasting a contract interface in FireFly: a `name`, a `version`, a list of `methods`, and a list of `events`. A `namespace` field will also be filled in automatically based on the URL path parameter. Here is an example of the structure of the required fields:

```json
{
    "name": "example",
    "version": "v1.0.0",
    "methods": [],
    "events": []
}
```

> **NOTE**: Contract interfaces are scoped to a namespace. Within a namespace each contract interface must have a unique name and version combination. The same name and version combination can exist in *different* namespaces simultaneously.

## Method

Let's look at a what goes inside the `methods` array now. It is also a JSON object that has a `name`, a list of `params` which are the arguments the function will take and a list of `returns` which are the return values of the function. Optionally, it also has a `description` which can be helpful in OpenAPI Spec generation.

```json
{
    "name": "add",
    "description": "Add two numbers together",
    "params": [],
    "returns": []
}
```

## Event

What goes into the `events` array is very similar. It is also a JSON object that has a `name` and a list of `params`. The difference is that `events` don't have `returns`. Arguments that are passed to the event when it is emitted are in `params`. Optionally, it also has a `description` which can be helpful in OpenAPI Spec generation.

```json
{
    "name": "added",
    "description": "An event that occurs when numbers have been added", 
    "params": []
}
```

## Param

Both `methods`, and `events` have lists of `params` or `returns`, and the type of JSON object that goes in each of these arrays is the same. It is simply a JSON object with a `name` and a `schema`. There is also an optional `details` field that is passed to the blockchain plugin for blockchain specific requirements.

```json
{
    "name": "x",
    "schema": {
        "type": "integer",
        "details": {}
    }
}
```

### Schema

The param `schema` is an important field which tells FireFly the type information about this particular field. This is used in several different places, such as OpenAPI Spec generation, API request validation, and blockchain request preparation.

The `schema` field accepts [JSON Schema (version 2020-12)](https://json-schema.org/specification-links.html#2020-12) with several additional requirements:

- A `type` field is always mandatory
- The list of valid types is:
    - `boolean`
    - `integer`
    - `string`
    - `object`
    - `array`
- Blockchain plugins can add their own specific requirements to this list of validation rules

> **NOTE**: Floats or decimals are not currently accepted because certain underlying blockchains (e.g. Ethereum) only allow integers

The type field here is the JSON input type when making a request to FireFly to invoke or query a smart contract. This type can be different from the actual blockchain type, usually specified in the `details` field, if there is a compatible type mapping between the two.

### Schema details

The details field is quite important in some cases. Because the `details` field is passed to the blockchain plugin, it is used to encapsulate blockchain specific type information about a particular field. Additionally, because each blockchain plugin can add rules to the list of schema requirements above, a blockchain plugin can enforce that certain fields are always present within the `details` field. 

For example, the Ethereum plugin always needs to know what Solidity type the field is. It also defines several optional fields. A full Ethereum details field may look like:

```json
{
    "type": "uint256",
    "internalType": "uint256",
    "indexed": false
}
```

## Automated generation of FireFly Interfaces

A convenience endpoint exists on the API to facilitate converting from native blockchain interface formats such as an Ethereum ABI to the FireFly Interface format. For details, please see the [API documentation for the contract interface generation endpoint](../swagger/swagger.html#/default/postGenerateContractInterface).

For an example of using this endpoint with a specific Ethereum contract, please see the [Getting Started guide to Work with custom smart contracts](../gettingstarted/custom_contracts.html#the-firefly-interface-format).

## Full Example

Putting it all together, here is a full example of the FireFly Interface format with all the fields filled in:

```json
{
    "namespace": "default",
    "name": "SimpleStorage",
    "description": "A simple smart contract that stores and retrieves an integer on-chain",
    "version": "v1.0.0",
    "methods": [
        {
            "name": "get",
            "description": "Retrieve the value of the stored integer",
            "params": [],
            "returns": [
                {
                    "name": "output",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "uint256",
                            "internalType": "uint256"
                        }
                    }
                }
            ]
        },
        {
            "name": "set",
            "description": "Set the stored value on-chain",
            "params": [
                {
                    "name": "newValue",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "uint256",
                            "internalType": "uint256"
                        }
                    }
                }
            ],
            "returns": []
        }
    ],
    "events": [
        {
            "name": "Changed",
            "description": "An event that is fired when the stored integer value changes",
            "params": [
                {
                    "name": "from",
                    "schema": {
                        "type": "string",
                        "details": {
                            "type": "address",
                            "internalType": "address",
                            "indexed": true
                        }
                    }
                },
                {
                    "name": "value",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "uint256",
                            "internalType": "uint256"
                        }
                    }
                }
            ]
        }
    ]
}
```