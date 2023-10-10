---
layout: default
title: Tezos
parent: pages.custom_smart_contracts
grand_parent: pages.tutorials
nav_order: 3
---

# Work with Tezos smart contracts

{: .no_toc }
This guide describes the steps to deploy a smart contract to a Tezos blockchain and use FireFly to interact with it in order to submit transactions, query for states and listening for events.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Smart Contract Languages

Smart contracts on Tezos can be programmed using familiar, developer-friendly languages. All features available on Tezos can be written in any of the high-level languages used to write smart contracts, such as Archetype, LIGO, and SmartPy. These languages all compile down to [Michelton](https://tezos.gitlab.io/active/michelson.html) and you can switch between languages based on your preferences and projects.

> **NOTE:** For this tutorial we are going to use [SmartPy](https://smartpy.io/) for building Tezos smart contracts utilizing the broadly adopted Python language.

## Example smart contract

First let's look at a simple contract smart contract called `SimpleStorage`, which we will be using on a Tezos blockchain. Here we have one state variable called 'storedValue' and initialized with the value 12. During initialization the type of the variable was defined as 'int'. You can see more at [SmartPy types](https://smartpy.io/manual/syntax/integers-and-mutez). And then we added a simple test, which set the storage value to 15 and checks that the value was changed as expected.

> **NOTE:** Tests are used to verify the validity of contract entrypoints and do not affect the state of the contract during deployment.

Here is the source for this contract:

```smarty
import smartpy as sp

@sp.module
def main():
    class SimpleStorage(sp.Contract):
        def __init__(self, value):
            self.data.storedValue = value

        @sp.entrypoint
        def replace(self, params):
            self.data.storedValue = params.value

@sp.add_test(name="SimpleStorage")
def test():
    c1 = main.SimpleStorage(12)
    scenario = sp.test_scenario(main)
    scenario.h1("SimpleStorage")
    scenario += c1
    c1.replace(value=15)
    scenario.verify(c1.data.storedValue == 15)
```

## Contract deployment

To deploy the contract, we will use [SmartPy IDE](https://smartpy.io/ide).
1. Open an IDE;
2. Paste the contract code;
3. Click "Run code" button;
4. Then you will see "Deploy Michelson Contract" button, click on that;
5. Choose the Ghostnet network;
6. Select an account, which you're going to use to deploy the contract;
7. Click "Estimate Cost From RPC" button;
8. Click "Deploy Contract" button;

![ContractDeployment](images/tezos_contract_deployment.png)
![ContractDeployment2](images/tezos_contract_deployment2.png)

Here we can see that our new contract address is `KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s`. This is the address that we will reference in the rest of this guide.

## The FireFly Interface Format

As we know from the previous section - smart contracts on the Tezos blockchain are using the domain-specific, stack-based programming language called [Michelton](https://tezos.gitlab.io/active/michelson.html). It is a key component of the Tezos platform and plays a fundamental role in defining the behavior of smart contracts and facilitating their execution.
This language is very efficient but also a bit tricky and challenging for learning, so in order to teach FireFly how to interact with the smart contract, we will be using [FireFly Interface (FFI)](../../reference/firefly_interface_format.md) to define the contract inteface which later will be encoded to Michelton.

The following FFI sample demonstrates the specification for the widely used FA2 (analogue of ERC721 for EVM) smart contract:

```json
{
    "namespace": "default",
    "name": "fa2",
    "version": "v1.0.0",
    "description": "",
    "methods": [
        {
            "name": "burn",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "token_ids",
                    "schema": {
                        "type": "array",
                        "details": {
                            "type": "nat",
                            "internalType": "nat"
                        }
                    }
                }
            ],
            "returns": []
        },
        {
            "name": "destroy",
            "pathname": "",
            "description": "",
            "params": [],
            "returns": []
        },
        {
            "name": "mint",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "owner",
                    "schema": {
                        "type": "string",
                        "details": {
                            "type": "address",
                            "internalType": "address"
                        }
                    }
                },
                {
                    "name": "requests",
                    "schema": {
                        "type": "array",
                        "details": {
                            "type": "schema",
                            "internalSchema": {
                                "type": "struct",
                                "args": [
                                    {
                                        "name": "metadata",
                                        "type": "bytes"
                                    },
                                    {
                                        "name": "token_id",
                                        "type": "nat"
                                    }
                                ]
                            }
                        }
                    }
                }
            ],
            "returns": []
        },
        {
            "name": "pause",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "pause",
                    "schema": {
                        "type": "boolean",
                        "details": {
                            "type": "boolean",
                            "internalType": "boolean"
                        }
                    }
                }
            ],
            "returns": []
        },
        {
            "name": "select",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "batch",
                    "schema": {
                        "type": "array",
                        "details": {
                            "type": "schema",
                            "internalSchema": {
                                "type": "struct",
                                "args": [
                                    {
                                        "name": "token_id",
                                        "type": "nat"
                                    },
                                    {
                                        "name": "recipient",
                                        "type": "address"
                                    },
                                    {
                                        "name": "token_id_start",
                                        "type": "nat"
                                    },
                                    {
                                        "name": "token_id_end",
                                        "type": "nat"
                                    }
                                ]
                            }
                        }
                    }
                }
            ],
            "returns": []
        },
        {
            "name": "transfer",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "batch",
                    "schema": {
                        "type": "array",
                        "details": {
                            "type": "schema",
                            "internalSchema": {
                                "type": "struct",
                                "args": [
                                    {
                                        "name": "from_",
                                        "type": "address"
                                    },
                                    {
                                        "name": "txs",
                                        "type": "list",
                                        "args": [
                                            {
                                                "type": "struct",
                                                "args": [
                                                    {
                                                        "name": "to_",
                                                        "type": "address"
                                                    },
                                                    {
                                                        "name": "token_id",
                                                        "type": "nat"
                                                    },
                                                    {
                                                        "name": "amount",
                                                        "type": "nat"
                                                    }
                                                ]
                                            }
                                        ]
                                    }
                                ]
                            }
                        }
                    }
                }
            ],
            "returns": []
        },
        {
            "name": "update_admin",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "admin",
                    "schema": {
                        "type": "string",
                        "details": {
                            "type": "address",
                            "internalType": "address"
                        }
                    }
                }
            ],
            "returns": []
        },
        {
            "name": "update_operators",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "requests",
                    "schema": {
                        "type": "array",
                        "details": {
                            "type": "schema",
                            "internalSchema": {
                                "type": "variant",
                                "variants": [
                                    "add_operator",
                                    "remove_operator"
                                ],
                                "args": [
                                    {
                                        "type": "struct",
                                        "args": [
                                            {
                                                "name": "owner",
                                                "type": "address"
                                            },
                                            {
                                                "name": "operator",
                                                "type": "address"
                                            },
                                            {
                                                "name": "token_id",
                                                "type": "nat"
                                            }
                                        ]
                                    }
                                ]
                            }
                        }
                    }
                }
            ],
            "returns": []
        }
    ],
    "events": []
}
```


## Broadcast the contract interface

Now that we have a FireFly Interface representation of our smart contract, we want to broadcast that to the entire network. This broadcast will be pinned to the blockchain, so we can always refer to this specific name and version, and everyone in the network will know exactly which contract interface we are talking about.

We will use the FFI JSON constructed above and `POST` that to the `/contracts/interfaces` API endpoint.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/contracts/interfaces`

```json
{
    "namespace": "default",
    "name": "simplestorage",
    "version": "v1.0.0",
    "description": "",
    "methods": [
        {
            "name": "replace",
            "pathname": "",
            "description": "",
            "params": [
                {
                    "name": "newValue",
                    "schema": {
                        "type": "integer",
                        "details": {
                            "type": "integer",
                            "internalType": "integer"
                        }
                    }
                }
            ],
            "returns": []
        }
    ],
    "events": []
}
```

### Response

```json
{
  "id": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2",
  "namespace": "default",
  "name": "simplestorage",
  "description": "",
  "version": "v1.0.0",
  "methods": [
    {
      "id": "6f707105-d8b5-4808-a864-51475086608d",
      "interface": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2",
      "name": "replace",
      "namespace": "default",
      "pathname": "replace",
      "description": "",
      "params": [
        {
          "name": "newValue",
          "schema": {
            "type": "integer",
            "details": {
              "type": "integer",
              "internalType": "integer"
            }
          }
        }
      ],
      "returns": []
    }
  ]
}
```

> **NOTE**: We can broadcast this contract interface conveniently with the help of FireFly Sandbox running at `http://127.0.0.1:5108`
* Go to the `Contracts Section`
* Click on `Define a Contract Interface`
* Select `FFI - FireFly Interface` in the `Interface Fromat` dropdown
* Copy the `FFI JSON` crafted by you into the `Schema` Field
* Click on `Run`

## Create an HTTP API for the contract

Now comes the fun part where we see some of the powerful, developer-friendly features of FireFly. The next thing we're going to do is tell FireFly to build an HTTP API for this smart contract, complete with an OpenAPI Specification and Swagger UI. As part of this, we'll also tell FireFly where the contract is on the blockchain.

Like the interface broadcast above, this will also generate a broadcast which will be pinned to the blockchain so all the members of the network will be aware of and able to interact with this API.

We need to copy the `id` field we got in the response from the previous step to the `interface.id` field in the request body below. We will also pick a name that will be part of the URL for our HTTP API, so be sure to pick a name that is URL friendly. In this case we'll call it `simple-storage`. Lastly, in the `location.address` field, we're telling FireFly where an instance of the contract is deployed on-chain.

> **NOTE**: The `location` field is optional here, but if it is omitted, it will be required in every request to invoke or query the contract. This can be useful if you have multiple instances of the same contract deployed to different addresses.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis`

```json
{
  "name": "simple-storage",
  "interface": {
    "id": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2"
  },
  "location": {
    "address": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
  }
}
```

### Response

```json
{
  "id": "af09de97-741d-4f61-8d30-4db5e7460f76",
  "namespace": "default",
  "interface": {
    "id": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2"
  },
  "location": {
    "address": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
  },
  "name": "simple-storage",
  "urls": {
    "openapi": "http://127.0.0.1:5000/api/v1/namespaces/default/apis/simple-storage/api/swagger.json",
    "ui": "http://127.0.0.1:5000/api/v1/namespaces/default/apis/simple-storage/api"
  }
}
```

## View OpenAPI spec for the contract

You'll notice in the response body that there are a couple of URLs near the bottom. If you navigate to the one labeled `ui` in your browser, you should see the Swagger UI for your smart contract.

![Swagger UI](images/simple_storage_swagger.png "Swagger UI")

## Invoke the smart contract

Now that we've got everything set up, it's time to use our smart contract! We're going to make a `POST` request to the `invoke/replace` endpoint to set the integer value on-chain. Let's set it to the value of `3` right now.

### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis/simple-storage/invoke/replace`

```json
{
  "input": {
    "newValue": 3
  },
  "key": "tz1cuFw1E2Mn2bVS8q8d7QoCb6FXC18JivSp"
}
```

> **NOTE**: The `key` field is the tezos account address, which will be used for signing our transactions.
See more at [transaction signing service set up](../chains/tezos_testnet.md#signatory).

### Response

```json
{
  "id": "cb38a538-7093-4150-8a80-6097a666df82",
  "namespace": "default",
  "tx": "5860befb-9f76-4aa0-a67c-55718b2c46d6",
  "type": "blockchain_invoke",
  "status": "Pending",
  "plugin": "tezos",
  "input": {
    "input": {
      "newValue": 3
    },
    "interface": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2",
    "key": "tz1cuFw1E2Mn2bVS8q8d7QoCb6FXC18JivSp",
    "location": {
      "address": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
    },
    "method": {
      "description": "",
      "id": "6f707105-d8b5-4808-a864-51475086608d",
      "interface": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2",
      "name": "replace",
      "namespace": "default",
      "params": [
        {
          "name": "newValue",
          "schema": {
            "details": {
              "internalType": "integer",
              "type": "integer"
            },
            "type": "integer"
          }
        }
      ],
      "pathname": "replace",
      "returns": []
    },
    "methodPath": "replace",
    "options": null,
    "type": "invoke"
  },
  "created": "2023-09-27T09:12:24.033724927Z",
  "updated": "2023-09-27T09:12:24.033724927Z"
}
```

You'll notice that we got an ID back with status `Pending`, and that's expected due to the asynchronous programming model of working with custom onchain logic in FireFly. After a while, let's see the result of our operation.

## Get the operation result

To see the result of the operation, call `/operations` endpoint with the operation ID from the previous step.

### Request

`GET` `http://localhost:5000/api/v1/operations/cb38a538-7093-4150-8a80-6097a666df82?fetchstatus=true`

### Response

```json
{
  "id": "cb38a538-7093-4150-8a80-6097a666df82",
  "namespace": "default",
  "tx": "5860befb-9f76-4aa0-a67c-55718b2c46d6",
  "type": "blockchain_invoke",
  "status": "Succeeded",
  "plugin": "tezos",
  "input": {
    "input": {
      "newValue": 3
    },
    "interface": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2",
    "key": "tz1cuFw1E2Mn2bVS8q8d7QoCb6FXC18JivSp",
    "location": {
      "address": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
    },
    "method": {
      "description": "",
      "id": "6f707105-d8b5-4808-a864-51475086608d",
      "interface": "c655704a-f0e2-4aa3-adbb-c7bf3280cdc2",
      "name": "replace",
      "namespace": "default",
      "params": [
        {
          "name": "newValue",
          "schema": {
            "details": {
              "internalType": "integer",
              "type": "integer"
            },
            "type": "integer"
          }
        }
      ],
      "pathname": "replace",
      "returns": []
    },
    "methodPath": "replace",
    "options": null,
    "type": "invoke"
  },
  "output": {
    "contractLocation": {
      "address": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
    },
    "headers": {
      "requestId": "default:cb38a538-7093-4150-8a80-6097a666df82",
      "type": "TransactionSuccess"
    },
    "protocolId": "PtNairobiyssHuh87hEhfVBGCVrK3WnS8Z2FT4ymB5tAa4r1nQf",
    "transactionHash": "opMjGX58akxboipsxMcTv5yc5M4Y2ZCGktos4E26zgEpgtHop7g"
  },
  "detail": {
    "receipt": {
      "blockHash": "BLy9BdEjBvHvhYkt8tR4wTQzHagCUmweh8K8uM6X5gXzPLbCmzP",
      "blockNumber": "4012016",
      "contractLocation": {
        "address": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
      },
      "extraInfo": [
        {
          "consumedGas": "1279",
          "contractAddress": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s",
          "counter": "18602183",
          "errorMessage": null,
          "fee": "404",
          "from": "tz1cuFw1E2Mn2bVS8q8d7QoCb6FXC18JivSp",
          "gasLimit": "1380",
          "paidStorageSizeDiff": "0",
          "status": "applied",
          "storage": "3",
          "to": "KT1D254HTPKq5GZNVcF73XBinG9BLybHqu8s"
        }
      ],
      "protocolId": "PtNairobiyssHuh87hEhfVBGCVrK3WnS8Z2FT4ymB5tAa4r1nQf",
      "success": true,
      "transactionIndex": "0"
    },
    "status": "Succeeded"
  }
}
```

Here we can see `detail.receipt.extraInfo.storage` section, which displays the latest state of the contract storage state after invoking the operation and that the value of the `storage` variable was changed to `3`.
