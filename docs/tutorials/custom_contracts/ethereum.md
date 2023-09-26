---
layout: default
title: Ethereum
parent: pages.custom_smart_contracts
grand_parent: pages.tutorials
nav_order: 1
---

# Work with Ethereum smart contracts

{: .no_toc }
This guide describes the steps to deploy a smart contract to an Ethereum blockchain and use FireFly to interact with it in order to submit transactions, query for states and listening for events.

> **NOTE:** This guide assumes that you are running a local FireFly stack with at least 2 members and an Ethereum blockchain created by the FireFly CLI. If you need help getting that set up, please see the [Getting Started guide to Start your environment](../../gettingstarted/setup_env.html).

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Example smart contract

For this tutorial, we will be using a well known, but slightly modified smart contract called `SimpleStorage`, and will be using this contract on an Ethereum blockchain. As the name implies, it's a very simple contract which stores an unsigned 256 bit integer, emits and event when the value is updated, and allows you to retrieve the current value.

Here is the source for this contract:

```solidity
// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.10;

// Declares a new contract
contract SimpleStorage {
    // Storage. Persists in between transactions
    uint256 x;

    // Allows the unsigned integer stored to be changed
    function set(uint256 newValue) public {
        x = newValue;
        emit Changed(msg.sender, newValue);
    }

    // Returns the currently stored unsigned integer
    function get() public view returns (uint256) {
        return x;
    }

    event Changed(address indexed from, uint256 value);
}
```

## Contract deployment

If you need to deploy an Ethereum smart contract with a signing key that FireFly will use for submitting future transactions it is recommended to use FireFly's built in contract deployment API. This is useful in many cases. For example, you may want to deploy a token contract and have FireFly mint some tokens. Many token contracts only allow the contract deployer to mint, so the contract would need to be deployed with a FireFly signing key.

You will need compile the contract yourself using [solc](https://docs.soliditylang.org/en/latest/installing-solidity.html) or some other tool. After you have compiled the contract, look in the JSON output file for the fields to build the request below.

### Request

| Field        | Description                                                                                                             |
| ------------ | ----------------------------------------------------------------------------------------------------------------------- |
| `key`        | The signing key to use to dpeloy the contract. If omitted, the namespaces's default signing key will be used.           |
| `contract`   | The compiled bytecode for your smart contract. It should be either a hex encded string or Base64.                       |
| `definition` | The full ABI JSON array from your compiled JSON file. Copy the entire value of the `abi` field from the `[` to the `]`. |
| `input`      | An ordered list of constructor arguments. Some contracts may not require any (such as this example).                    |

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/deploy`

```json
{
  "contract": "608060405234801561001057600080fd5b5061019e806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806360fe47b11461003b5780636d4ce63c14610057575b600080fd5b61005560048036038101906100509190610111565b610075565b005b61005f6100cd565b60405161006c919061014d565b60405180910390f35b806000819055503373ffffffffffffffffffffffffffffffffffffffff167fb52dda022b6c1a1f40905a85f257f689aa5d69d850e49cf939d688fbe5af5946826040516100c2919061014d565b60405180910390a250565b60008054905090565b600080fd5b6000819050919050565b6100ee816100db565b81146100f957600080fd5b50565b60008135905061010b816100e5565b92915050565b600060208284031215610127576101266100d6565b5b6000610135848285016100fc565b91505092915050565b610147816100db565b82525050565b6000602082019050610162600083018461013e565b9291505056fea2646970667358221220e6cbd7725b98b234d07bc1823b60ac065b567c6645d15c8f8f6986e5fa5317c664736f6c634300080b0033",
  "definition": [
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "from",
          "type": "address"
        },
        {
          "indexed": false,
          "internalType": "uint256",
          "name": "value",
          "type": "uint256"
        }
      ],
      "name": "Changed",
      "type": "event"
    },
    {
      "inputs": [],
      "name": "get",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "newValue",
          "type": "uint256"
        }
      ],
      "name": "set",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ],
  "input": []
}
```

### Response

```json
{
  "id": "aa155a3c-2591-410e-bc9d-68ae7de34689",
  "namespace": "default",
  "tx": "4712ffb3-cc1a-4a91-aef2-206ac068ba6f",
  "type": "blockchain_deploy",
  "status": "Succeeded",
  "plugin": "ethereum",
  "input": {
    "contract": "608060405234801561001057600080fd5b5061019e806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806360fe47b11461003b5780636d4ce63c14610057575b600080fd5b61005560048036038101906100509190610111565b610075565b005b61005f6100cd565b60405161006c919061014d565b60405180910390f35b806000819055503373ffffffffffffffffffffffffffffffffffffffff167fb52dda022b6c1a1f40905a85f257f689aa5d69d850e49cf939d688fbe5af5946826040516100c2919061014d565b60405180910390a250565b60008054905090565b600080fd5b6000819050919050565b6100ee816100db565b81146100f957600080fd5b50565b60008135905061010b816100e5565b92915050565b600060208284031215610127576101266100d6565b5b6000610135848285016100fc565b91505092915050565b610147816100db565b82525050565b6000602082019050610162600083018461013e565b9291505056fea2646970667358221220e6cbd7725b98b234d07bc1823b60ac065b567c6645d15c8f8f6986e5fa5317c664736f6c634300080b0033",
    "definition": [
      {
        "anonymous": false,
        "inputs": [
          {
            "indexed": true,
            "internalType": "address",
            "name": "from",
            "type": "address"
          },
          {
            "indexed": false,
            "internalType": "uint256",
            "name": "value",
            "type": "uint256"
          }
        ],
        "name": "Changed",
        "type": "event"
      },
      {
        "inputs": [],
        "name": "get",
        "outputs": [
          {
            "internalType": "uint256",
            "name": "",
            "type": "uint256"
          }
        ],
        "stateMutability": "view",
        "type": "function"
      },
      {
        "inputs": [
          {
            "internalType": "uint256",
            "name": "newValue",
            "type": "uint256"
          }
        ],
        "name": "set",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
      }
    ],
    "input": [],
    "key": "0xddd93a452bfc8d3e62bbc60c243046e4d0cb971b",
    "options": null
  },
  "output": {
    "headers": {
      "requestId": "default:aa155a3c-2591-410e-bc9d-68ae7de34689",
      "type": "TransactionSuccess"
    },
    "contractLocation": {
      "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
    },
    "protocolId": "000000000024/000000",
    "transactionHash": "0x32d1144091877266d7f0426e48db157e7d1a857c62e6f488319bb09243f0f851"
  },
  "created": "2023-02-03T15:42:52.750277Z",
  "updated": "2023-02-03T15:42:52.750277Z"
}
```

Here we can see in the response above under the `output` section that our new contract address is `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1`. This is the address that we will reference in the rest of this guide.

## The FireFly Interface Format

If you have an Ethereum ABI for an existing smart contract, there is an HTTP endpoint on the FireFly API that will take the ABI as input and automatically generate the FireFly Interface for you. Rather than handcrafting our FFI, we'll let FireFly generate it for us using that endpoint now.

### Request

Here we will take the JSON ABI generated by `truffle` or `solc` and `POST` that to FireFly to have it automatically generate the FireFly Interface for us. Copy the `abi` from the compiled JSON file, and put that inside an `input` object like the example below:

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/interfaces/generate`

```json
{
  "input": {
    "abi": [
      {
        "anonymous": false,
        "inputs": [
          {
            "indexed": true,
            "internalType": "address",
            "name": "from",
            "type": "address"
          },
          {
            "indexed": false,
            "internalType": "uint256",
            "name": "value",
            "type": "uint256"
          }
        ],
        "name": "Changed",
        "type": "event"
      },
      {
        "inputs": [],
        "name": "get",
        "outputs": [
          {
            "internalType": "uint256",
            "name": "",
            "type": "uint256"
          }
        ],
        "stateMutability": "view",
        "type": "function"
      },
      {
        "inputs": [
          {
            "internalType": "uint256",
            "name": "newValue",
            "type": "uint256"
          }
        ],
        "name": "set",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
      }
    ]
  }
}
```

### Response

FireFly generates and returns the the full FireFly Interface for the SimpleStorage contract in the response body:

```json
{
  "namespace": "default",
  "name": "",
  "description": "",
  "version": "",
  "methods": [
    {
      "name": "get",
      "pathname": "",
      "description": "",
      "params": [],
      "returns": [
        {
          "name": "",
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
      "pathname": "",
      "description": "",
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
      "description": "",
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

## Broadcast the contract interface

Now that we have a FireFly Interface representation of our smart contract, we want to broadcast that to the entire network. This broadcast will be pinned to the blockchain, so we can always refer to this specific name and version, and everyone in the network will know exactly which contract interface we are talking about.

We will take the output from the previous HTTP response above, **fill in the name and version** and then `POST` that to the `/contracts/interfaces` API endpoint.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/interfaces`

```json
{
  "namespace": "default",
  "name": "SimpleStorage",
  "version": "v1.0.0",
  "description": "",
  "methods": [
    {
      "name": "get",
      "pathname": "",
      "description": "",
      "params": [],
      "returns": [
        {
          "name": "",
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
      "pathname": "",
      "description": "",
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
      "description": "",
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

### Response

```json
{
  "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3",
  "message": "3cd0dde2-1e39-4c9e-a4a1-569e87cca93a",
  "namespace": "default",
  "name": "SimpleStorage",
  "description": "",
  "version": "v1.0.0",
  "methods": [
    {
      "id": "56467890-5713-4463-84b8-4537fcb63d8b",
      "contract": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3",
      "name": "get",
      "namespace": "default",
      "pathname": "get",
      "description": "",
      "params": [],
      "returns": [
        {
          "name": "",
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
      "id": "6b254d1d-5f5f-491e-bbd2-201e96892e1a",
      "contract": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3",
      "name": "set",
      "namespace": "default",
      "pathname": "set",
      "description": "",
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
      "id": "aa1fe67b-b2ac-41af-a7e7-7ad54a30a78d",
      "contract": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3",
      "namespace": "default",
      "pathname": "Changed",
      "name": "Changed",
      "description": "",
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

## Create an HTTP API for the contract

Now comes the fun part where we see some of the powerful, developer-friendly features of FireFly. The next thing we're going to do is tell FireFly to build an HTTP API for this smart contract, complete with an OpenAPI Specification and Swagger UI. As part of this, we'll also tell FireFly where the contract is on the blockchain. Like the interface broadcast above, this will also generate a broadcast which will be pinned to the blockchain so all the members of the network will be aware of and able to interact with this API.

We need to copy the `id` field we got in the response from the previous step to the `interface.id` field in the request body below. We will also pick a name that will be part of the URL for our HTTP API, so be sure to pick a name that is URL friendly. In this case we'll call it `simple-storage`. Lastly, in the `location.address` field, we're telling FireFly where an instance of the contract is deployed on-chain.

> **NOTE**: The `location` field is optional here, but if it is omitted, it will be required in every request to invoke or query the contract. This can be useful if you have multiple instances of the same contract deployed to different addresses.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis`

```json
{
  "name": "simple-storage",
  "interface": {
    "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
  },
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  }
}
```

### Response

```json
{
  "id": "9a681ec6-1dee-42a0-b91b-61d23a814b0f",
  "namespace": "default",
  "interface": {
    "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
  },
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "name": "simple-storage",
  "message": "d90d0386-8874-43fb-b7d3-485c22f35f47",
  "urls": {
    "openapi": "http://127.0.0.1:5000/api/v1/namespaces/default/apis/simple-storage/api/swagger.json",
    "ui": "http://127.0.0.1:5000/api/v1/namespaces/default/apis/simple-storage/api"
  }
}
```

## View OpenAPI spec for the contract

You'll notice in the response body that there are a couple of URLs near the bottom. If you navigate to the one labeled `ui` in your browser, you should see the Swagger UI for your smart contract.

![Swagger UI](../../images/simple_storage_swagger.png "Swagger UI")

## Invoke the smart contract

Now that we've got everything set up, it's time to use our smart contract! We're going to make a `POST` request to the `invoke/set` endpoint to set the integer value on-chain. Let's set it to the value of `3` right now.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis/simple-storage/invoke/set`

```json
{
  "input": {
    "newValue": 3
  }
}
```

### Response

```json
{
  "id": "41c67c63-52cf-47ce-8a59-895fe2ffdc86"
}
```

You'll notice that we just get an ID back here, and that's expected due to the asynchronous programming model of working with smart contracts in FireFly. To see what the value is now, we can query the smart contract. In a little bit, we'll also subscribe to the events emitted by this contract so we can know when the value is updated in realtime.

## Query the current value

To make a read-only request to the blockchain to check the current value of the stored integer, we can make a `POST` to the `query/get` endpoint.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis/simple-storage/query/get`

```json
{}
```

### Response

```json
{
  "output": "3"
}
```

> **NOTE:** Some contracts may have queries that require input parameters. That's why the query endpoint is a `POST`, rather than a `GET` so that parameters can be passed as JSON in the request body. This particular function does not have any parameters, so we just pass an empty JSON object.

## Passing additional options with a request

Some smart contract functions may accept or require additional options to be passed with the request. For example, a Solidity function might be `payable`, meaning that a `value` field must be specified, indicating an amount of ETH to be transferred with the request. Each of your smart contract API's `/invoke` or `/query` endpoints support an `options` object in addition to the `input` arguments for the function itself.

Here is an example of sending 100 wei with a transaction:

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis/simple-storage/invoke/set`

```json
{
  "input": {
    "newValue": 3
  },
  "options": {
    "value": 100
  }
}
```

### Response

```json
{
  "id": "41c67c63-52cf-47ce-8a59-895fe2ffdc86"
}
```

## Create a blockchain event listener

Now that we've seen how to submit transactions and preform read-only queries to the blockchain, let's look at how to receive blockchain events so we know when things are happening in realtime.

If you look at the source code for the smart contract we're working with above, you'll notice that it emits an event when the stored value of the integer is set. In order to receive these events, we first need to instruct FireFly to listen for this specific type of blockchain event. To do this, we create an **Event Listener**. The `/contracts/listeners` endpoint is RESTful so there are `POST`, `GET`, and `DELETE` methods available on it. To create a new listener, we will make a `POST` request. We are going to tell FireFly to listen to events with name `"Changed"` from the FireFly Interface we defined earlier, referenced by its ID. We will also tell FireFly which contract address we expect to emit these events, and the topic to assign these events to. Topics are a way for applications to subscribe to events they are interested in.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/listeners`

```json
{
  "interface": {
    "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
  },
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "eventPath": "Changed",
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```

### Response

```json
{
  "id": "1bfa3b0f-3d90-403e-94a4-af978d8c5b14",
  "interface": {
    "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
  },
  "namespace": "default",
  "name": "sb-66209ffc-d355-4ac0-7151-bc82490ca9df",
  "protocolId": "sb-66209ffc-d355-4ac0-7151-bc82490ca9df",
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "created": "2022-02-17T22:02:36.34549538Z",
  "event": {
    "name": "Changed",
    "description": "",
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
  },
  "options": {
    "firstEvent": "oldest"
  }
}
```

We can see in the response, that FireFly pulls all the schema information from the FireFly Interface that we broadcasted earlier and creates the listener with that schema. This is useful so that we don't have to enter all of that data again.

### Querying listener status

If you are interested in learning about the current state of a listener you have created, you can query with the `fetchstatus` parameter. For FireFly stacks with an EVM compatible blockchain connector, the response will include checkpoint information and if the listener is currently in catchup mode.

#### Request / Response

`GET` `http://localhost:5000/api/v1/namespaces/default/contracts/listeners/1bfa3b0f-3d90-403e-94a4-af978d8c5b14?fetchstatus`

```json
{
  "id": "1bfa3b0f-3d90-403e-94a4-af978d8c5b14",
  "interface": {
    "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
  },
  "namespace": "default",
  "name": "sb-66209ffc-d355-4ac0-7151-bc82490ca9df",
  "protocolId": "sb-66209ffc-d355-4ac0-7151-bc82490ca9df",
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "created": "2022-02-17T22:02:36.34549538Z",
  "event": {
    "name": "Changed",
    "description": "",
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
  },
  "status": {
    "checkpoint": {
      "block": 0,
      "transactionIndex": -1,
      "logIndex": -1
    },
    "catchup": true
  },
  "options": {
    "firstEvent": "oldest"
  }
}
```

## Subscribe to events from our contract

Now that we've told FireFly that it should listen for specific events on the blockchain, we can set up a **Subscription** for FireFly to send events to our app. To set up our subscription, we will make a `POST` to the `/subscriptions` endpoint.

We will set a friendly name `simple-storage` to identify the Subscription when we are connecting to it in the next step.

We're also going to set up a filter to only send events blockchain events from our listener that we created in the previous step. To do that, we'll **copy the listener ID** from the step above (`1bfa3b0f-3d90-403e-94a4-af978d8c5b14`) and set that as the value of the `listener` field in the example below:

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/subscriptions`

```json
{
  "namespace": "default",
  "name": "simple-storage",
  "transport": "websockets",
  "filter": {
    "events": "blockchain_event_received",
    "blockchainevent": {
      "listener": "1bfa3b0f-3d90-403e-94a4-af978d8c5b14"
    }
  },
  "options": {
    "firstEvent": "oldest"
  }
}
```

### Response

```json
{
  "id": "f826269c-65ed-4634-b24c-4f399ec53a32",
  "namespace": "default",
  "name": "simple-storage",
  "transport": "websockets",
  "filter": {
    "events": "blockchain_event_received",
    "message": {},
    "transaction": {},
    "blockchainevent": {
      "listener": "1bfa3b0f-3d90-403e-94a4-af978d8c5b14"
    }
  },
  "options": {
    "firstEvent": "-1",
    "withData": false
  },
  "created": "2022-03-15T17:35:30.131698921Z",
  "updated": null
}
```

## Receive custom smart contract events

The last step is to connect a WebSocket client to FireFly to receive the event. You can use any WebSocket client you like, such as [Postman](https://www.postman.com/) or a command line app like [`websocat`](https://github.com/vi/websocat).

Connect your WebSocket client to `ws://localhost:5000/ws`.

After connecting the WebSocket client, send a message to tell FireFly to:

- Start sending events
- For the Subscription named `simple-storage`
- On the `default` namespace
- Automatically "ack" each event which will let FireFly immediately send the next event when available

```json
{
  "type": "start",
  "name": "simple-storage",
  "namespace": "default",
  "autoack": true
}
```

### WebSocket event

After creating the subscription, you should see an event arrive on the connected WebSocket client that looks something like this:

```json
{
  "id": "0f4a31d6-9743-4537-82df-5a9c76ccbd1e",
  "sequence": 24,
  "type": "blockchain_event_received",
  "namespace": "default",
  "reference": "dd3e1554-c832-47a8-898e-f1ee406bea41",
  "created": "2022-03-15T17:32:27.824417878Z",
  "blockchainevent": {
    "id": "dd3e1554-c832-47a8-898e-f1ee406bea41",
    "sequence": 7,
    "source": "ethereum",
    "namespace": "default",
    "name": "Changed",
    "listener": "1bfa3b0f-3d90-403e-94a4-af978d8c5b14",
    "protocolId": "000000000010/000000/000000",
    "output": {
      "from": "0xb7e6a5eb07a75a2c81801a157192a82bcbce0f21",
      "value": "3"
    },
    "info": {
      "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1",
      "blockNumber": "10",
      "logIndex": "0",
      "signature": "Changed(address,uint256)",
      "subId": "sb-724b8416-786d-4e67-4cd3-5bae4a26eb0e",
      "timestamp": "1647365460",
      "transactionHash": "0xd5b5c716554097b2868d8705241bb2189bb76d16300f702ad05b0b02fccc4afb",
      "transactionIndex": "0x0"
    },
    "timestamp": "2022-03-15T17:31:00Z",
    "tx": {
      "type": ""
    }
  },
  "subscription": {
    "id": "f826269c-65ed-4634-b24c-4f399ec53a32",
    "namespace": "default",
    "name": "simple-storage"
  }
}
```

You can see in the event received over the WebSocket connection, the blockchain event that was emitted from our first transaction, which happened in the past. We received this event, because when we set up both the Listener, and the Subscription, we specified the `"firstEvent"` as `"oldest"`. This tells FireFly to look for this event from the beginning of the blockchain, and that your app is interested in FireFly events since the beginning of FireFly's event history.

In the event, we can also see the `blockchainevent` itself, which has an `output` object. These are the `params` in our FireFly Interface, and the actual output of the event. Here we can see the `value` is `3` which is what we set the integer to in our original transaction.

### Subscription offset

If you query by the ID of your subscription with the `fetchstatus` parameter, you can see its current `offset`.

`GET` `http://localhost:5000/api/v1/namespaces/default/subscriptions/f826269c-65ed-4634-b24c-4f399ec53a32`

```json
{
  "id": "f826269c-65ed-4634-b24c-4f399ec53a32",
  "namespace": "default",
  "name": "simple-storage",
  "transport": "websockets",
  "filter": {
    "events": "blockchain_event_received",
    "message": {},
    "transaction": {},
    "blockchainevent": {
      "listener": "1bfa3b0f-3d90-403e-94a4-af978d8c5b14"
    }
  },
  "options": {
    "firstEvent": "-1",
    "withData": false
  },
  "status": {
    "offset": 20
  }
  "created": "2022-03-15T17:35:30.131698921Z",
  "updated": null
}
```

**You've reached the end of the main guide to working with custom smart contracts in FireFly**. Hopefully this was helpful and gives you what you need to get up and running with your own contracts. There are several additional ways to invoke or query smart contracts detailed below, so feel free to keep reading if you're curious.

## Appendix I: Work with a custom contract without creating a named API

FireFly aims to offer a developer-friendly and flexible approach to using custom smart contracts. The guide above has detailed the most robust and feature-rich way to use custom contracts with FireFly, but there are several alternative API usage patterns available as well.

It is possible to broadcast a contract interface and use a smart contract that implements that interface without also broadcasting a named API as above. There are several key differences (which may or may not be desirable) compared to the method outlined in the full guide above:

- OpenAPI Spec and Swagger UI are not available
- Each HTTP request to invoke/query the contract will need to include the contract location
- The contract location will _not_ have been broadcasted to all other members of the network
- The URL to invoke/query the contract will be different (described below)

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/interfaces/8bdd27a5-67c1-4960-8d1e-7aa31b9084d3/invoke/set`

```json
{
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "input": {
    "newValue": 7
  }
}
```

### Response

```json
{
  "id": "f310fa4a-73d8-4777-9f9d-dfa5012a052f"
}
```

All of the same invoke, query, and subscribe endpoints are available on the contract interface itself.

## Appendix II: Work directly with contracts with inline requests

The final way of working with custom smart contracts with FireFly is to just put everything FireFly needs all in one request, each time a contract is invoked or queried. This is the most lightweight, but least feature-rich way of using a custom contract.

To do this, we will need to put both the contract location, and a subset of the FireFly Interface that describes the method we want to invoke in the request body, in addition to the function input.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/invoke`

```json
{
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "method": {
    "name": "set",
    "params": [
      {
        "name": "x",
        "schema": {
          "type": "integer",
          "details": {
            "type": "uint256"
          }
        }
      }
    ],
    "returns": []
  },
  "input": {
    "x": 42
  }
}
```

### Response

```json
{
  "id": "386d3e23-e4bc-4a9b-bc1f-452f0a8c9ae5"
}
```
