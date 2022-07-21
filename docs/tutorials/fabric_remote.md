---
layout: default
title: Work with remote fabric network
parent: pages.tutorials
nav_order:
---

# Work with remote Hyperledger Fabric Network

{: .no_toc }

This guide describes the steps to use FireFly to interact with a chaincode deployed to an external Hyperledger Fabric blockchain network (not created by Firefly) in order to submit transactions and query for states.

> **NOTE:** This guide assumes that you are already running a Hyperledger Fabric blockchain network and have a chaicode deployed in that network that you want to connect to with FireFly.

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Creating a FireFly stack with Fabric provider

In the ff init command, pass fabric in the `-b`  flag to select fabric as the blockchain provider. You can also pass an additional `--prompt-names` flag to give actual names for your organizations in the firefly setup.

```
ff init -b fabric --prompt-names
```

Choose a stack name. For this guide, I will choose the name `my-remote-fabric`, but you can pick whatever you want.

```
stack name: my-remote-fabric
```

For this guide, we will be creating only one supernode for one of the organizations in our fabric netwok and using that organizational identity to interact with the chaincode. You can create multiple supernodes for different organizations in your fabric network by passing here the number of members in your stack.

```
number of members: 1
```

Next, you will be prompted to enter a name for your organization and node. If you selected more than 1 member for your stack in the previous step, it will prompt you to give name for each member. For this guide, I will choose the name `my-org` and `my-org-node`, but you can pick whatever you want.

```
name for org 0: my-org
name for node 0: my-org-node
```

At the end of it, you will be greeted with something like :

```
Stack 'my-remote-fabric' created!
To start your new stack run:

ff start my-remote-fabric
```

Next, you need to edit your stack configuration before you run it.

## Configuring the stack to connect to your remote Fabric network

Now that you have created a stack, let's open it up in your favourite code editor to configure it further. By default, the path where it gets created in your system is `$HOME/.firefly/stacks/my-remote-fabric.`

### Add msp folder representing your fabric identity

In `$HOME/.firefly/stacks/my-remote-fabric`, you need to add an msp folder. This msp folder essentially describes the fabric organizational identiy which will be interacting with your chaincode. Inside the msp folder, you must have all the certifactes and the private key to be used by firefly to query/invoke your chaincode.

You should be having this msp folder for an organizational identity from when you would have set up your fabric network. For more details, please refer to [Hyperledger Fabric Documentation for MSP](https://hyperledger-fabric.readthedocs.io/en/latest/msp.html).

Now, you need to mount this msp directory as a docker volume in your fabconnect serivce.

> **NOTE:** All docker-compose.yml related changes should be done in docker-compose.override.yml.

For this, simply copy the fabconnect_0 service as it is from your docker-compose.yml and paste it in your docker-compose.override.yml. Next, in your docker-compose.override.yml, in the volumes section of the fabconnect_0 service, you need to add a new docker volume mapping your host's msp directory to your fabconnect_0 docker container's /etc/firefly/msp directory. 

It will look something like :

```
-/Users/<username>/.firefly/stacks/my-remote-fabric/msp:/etc/firefly/msp
```

Your docker-compose.override.yml file after editing should look something like :

```
# Add custom config overrides here
# See https://docs.docker.com/compose/extends
version: "2.1"
services:
    fabconnect_0:
        container_name: my-remote-fabric_fabconnect_0
        image: ghcr.io/hyperledger/firefly-fabconnect@sha256:0dff97610990414d8ceb24859ced7712fa18b71b1d22b6a6f46fd51266413a22
        command: -f /fabconnect/fabconnect.yaml
        volumes:
            - fabconnect_receipts_0:/fabconnect/receipts
            - fabconnect_events_0:/fabconnect/events
            - /Users/<username>/.firefly/stacks/my-remote-fabric/runtime/blockchain/fabconnect.yaml:/fabconnect/fabconnect.yaml
            - /Users/<username>/.firefly/stacks/my-remote-fabric/runtime/blockchain/ccp.yaml:/fabconnect/ccp.yaml
            - /Users/<username>/.firefly/stacks/my-remote-fabric/msp:/etc/firefly/msp
            - firefly_fabric:/etc/firefly
        ports:
            - 5102:3000
        depends_on:
            fabric_ca:
                condition: service_started
            fabric_orderer:
                condition: service_started
            fabric_peer:
                condition: service_started
        healthcheck:
            test:
                - CMD
                - wget
                - -O
                - '-'
                - http://localhost:3000/status
        logging:
            driver: json-file
            options:
                max-file: "1"
                max-size: 10m
```

> **NOTE:** If you had multiple members in your stack, with each member representing a different fabric organizational entity, then you would need to add separate msp folders for all of them in a similar manner

### Edit your fabric connection profile (ccp.yml)

In `$HOME/.firefly/stacks/my-remote-fabric/init/blockchain`, you will find a ccp.yml file which is mounted as a volume in our fabconnect service. This file is also commonly referred to as fabric connection profile.

This ccp.yml file is written to connect to the default localhost containerized fabric network that is created by firefly CLI when you run your stack. You probably don't want to connect to that network since you are viewing this guide. So what you need to do is :

* You need to edit this ccp.yml file and give your fabric network and client details here instead.
* For the `credentialStore/cryptoStore` path or the `cryptoconfig` path in your ccp.yml, you need to provide the path of your msp folder in your fabconnect docker container : `/etc/firefly/msp`
* For the `certificates/key` paths in your ccp.yml, you need to provide the paths of corresponding key/certificates in your msp folder in your fabconnect docker container :`/etc/firefly/msp/<path-to-cert-or-key>`

Your ccp.yml file after editing should look something like :

```
certificateAuthorities:
    my-fabric-network-ca:
        tlsCACerts:
            path: /etc/firefly/msp/tlscacerts/tls-ca-cert.pem
        url: https://my-fabric-network-ca.my-domain.com:443
        registrar:
            enrollId: my-enroll-id
            enrollSecret: my-enroll-secret
channels:
    my-channel:
        orderers:
            - my-orderer-node
        peers:
            my-peer-node:
                chaincodeQuery: true
                endorsingPeer: true
                eventSource: true
                ledgerQuery: true
client:
    BCCSP:
        security:
            default:
                provider: SW
            enabled: true
            hashAlgorithm: SHA2
            level: 256
            softVerify: true
    credentialStore:
        cryptoStore:
            path: /etc/firefly/msp
        path: /etc/firefly/msp
    cryptoconfig:
        path: /etc/firefly/msp
    logging:
        level: info
    organization: my-org
    tlsCerts:
        client:
            cert:
                path: /etc/firefly/msp/signcerts/cert.pem
            key:
                path: /etc/firefly/msp/keystore/key.pem
orderers:
    my-orderer-node:
        tlsCACerts:
            path: /etc/firefly/msp/tlscacerts/tls-ca-cert.pem
        url: grpcs://my-orderer-node.my-domain.com:7050
organizations:
    my-org:
        certificateAuthorities:
            - my-fabric-network-ca
        cryptoPath: /etc/firefly/msp
        mspid: my-org-mspid
        peers:
            - my-peer-node
peers:
    my-peer-node:
        tlsCACerts:
            path: /etc/firefly/msp/tlscacerts/tls-ca-cert.pem
        url: grpcs://my-peer-node.my-domain.com:7051
version: 1.1.0%

```

> **NOTE:** If you had multiple members in your stack, with each member representing a different fabric organizational entity, then you would need to add separate ccp.yml files for all of them in a similar manner.

## Setting up Batchpin chaincode and Running your stack

### Deploying batchpin chaincode to fabric network

For Firefly to succeesfully integrate and interact with your remote fabric network, you need to also deploy a Firefly batchpin chaincode in your network setup. This chaincode can be found in the [FireFly git repo](https://github.com/hyperledger/firefly/tree/main/smart_contracts/fabric/firefly-go).

Once you have successfully packaged, installed, approved and commited this chaincode in your remote fabric network, carefully note down the chaincode name you gave and the channel name where this chaincode is deployed for future reference.

> **NOTE:** You can deploy the batchpin chaincode to the same channel where your application chaincode lives or to a separate channel as well.

### Update batchpin chaincode details in firefly core config

In `$HOME/.firefly/stacks/my-remote-fabric/init/config`, you will find a firefly_core_0.yml file which is used by firefly core service in our firefly network setup.

In this file, under the fabconnect section of blockchain-fabric section, you need to change the chaincode value to your batchpin chaincode name, and the channel value to the channel name where your batchpin chaincode was deployed in the previous step.

### Start your stack

To start your stack now, simply run:

```
ff start my-remote-fabric
```

This may take a minute or two and in the background the FireFly CLI will do the following for you:

- Download Docker images for all of the components of the Supernode
- Set up configuration between all the components
- Deploy FireFly's `BatchPin` smart contract to our remote fabric network
- Register an identity for each member and node in our remote fabric network
- It will probably also initialize a new blockchain and blockchain node running inside a container and also deploy an `ERC-1155` token smart contract to that local fabric network. This behaviour can be ignored.

After your stack finishes starting it will print out the links to each member's UI and the Sandbox for that node. For this guide, it's just one member.

```
Web UI for member '0': http://127.0.0.1:5000/ui
Sandbox UI for member '0': http://127.0.0.1:5108

To see logs for your stack run:

ff logs my-remote-fabric

```

## Define and Broadcast FireFly Interface Document for your chaincode

In order to teach FireFly how to interact with chaincode deployed in your remote fabric network, a FireFly Interface (FFI) document is needed. While Ethereum (or other EVM based blockchains) requires an Application Binary Interface (ABI) to govern the interaction between the client and the smart contract, which is specific to each smart contract interface design, Fabric defines a generic [chaincode interface](https://hyperledger-fabric.readthedocs.io/en/release-2.0/chaincode4ade.html#chaincode-api) and leaves the encoding and decoding of the parameter values to the discretion of the chaincode developer.

As a result, the FFI document for a Fabric chaincode must be hand-crafted. For more details on  it, you can refer to this [guide](https://hyperledger.github.io/firefly/tutorials/custom_contracts/fabric.html#the-firefly-interface-format).

Once you have a FireFly Interface representation of your chaincode, you need to broadcast that to the entire network. This broadcast will be pinned to the blockchain, so you can always refer to this specific name and version, and everyone in the network will know exactly which contract interface we are talking about.

We will be making this broadcast conveniently with the help of FireFly Sandbox running at `http://127.0.0.1:5108`

* Go to the `Contracts Section`
* Click on `Define a Contract Interface`
* Select `FFI - FireFly Interface` in the `Interface Fromat` dropdown
* Copy the `FFI JSON` created by you into the `Schema` Field
* Click on `Run`

This will broadcast your chaincode FFI to the entire firefly network, and also pin the broadcast to your blockchain.

## Create an HTTP API for the contract

Now comes the fun part where we see some of the powerful, developer-friendly features of FireFly. The next thing we're going to do is tell FireFly to build an HTTP API for this chaincode, complete with an OpenAPI Specification and Swagger UI. As part of this, we'll also tell FireFly where the chaincode is on the blockchain.

Like the interface broadcast above, this will also generate a broadcast which will be pinned to the blockchain so all the members of the network will be aware of and able to interact with this API.

We will be making this broadcast also conveniently with the help of FireFly Sandbox running at `http://127.0.0.1:5108`

* Go to the `Contracts Section`
* Click on `Register a Contract API`
* Select the name of your previously broadcasted FFI in the `Contract Interface` dropdown
* Input a name in the `Name` Field that will be part of the URL for our HTTP API
* In the `Chaincode` Field, give the name of your chaincode for which you wrote the FFI and want to interact with
* In the `Channel` Field, give the name of the channel of your fabric network where your chaincode is deployed
* Click on `Run`

This will broadcast your API to the entire firefly network, and also pin the broadcast to our blockchain.

Now if you go to FireFly UI running at `http://127.0.0.1:5000/ui`, under the APIs sub-section of the Blockchain section of sidebar, you will find the API details for your contract. In the UI column, click on the link button to go the Swagger UI.

From here, you can easily query/invoke all your chaincode methods with your organizational context.

### /invoke/\* endpoints

The `/invoke` endpoints in the generated API are for submitting transactions. These endpoints will be mapped to the `POST /transactions` endpoint of the [FabConnect API](https://github.com/hyperledger/firefly-fabconnect).

### /query/\* endpoints

The `/query` endpoints in the generated API, on the other hand, are for sending query requests. These endpoints will be mapped to the `POST /query` endpoint of the Fabconnect API, which under the cover only sends chaincode endorsement requests to the target peer node without sending a trasaction payload to the orderer node.

## Invoke/Query the chaincode

### Invoke

Now that we've got everything set up, it's time to use our chaincode!

> **NOTE:** Suppose the chaincode deployed in our remote fabric network is the classic `asset_transfer` chaincode.

We're going to make a `POST` request to the `invoke/CreateAsset` endpoint to create a new asset.

#### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis/asset_transfer/invoke/CreateAsset`

```json
{
  "input": {
    "color": "blue",
    "id": "asset-01",
    "owner": "Harry",
    "size": "30",
    "value": "23400"
  }
}
```

#### Response

```json
{
  "id": "b8e905cc-bc23-434a-af7d-13c6d85ae545",
  "namespace": "default",
  "tx": "79d2668e-4626-4634-9448-1b40fa0d9dfd",
  "type": "blockchain_invoke",
  "status": "Pending",
  "plugin": "fabric",
  "input": {
    "input": {
      "color": "blue",
      "id": "asset-02",
      "owner": "Harry",
      "size": "30",
      "value": "23400"
    },
    "interface": "f1e5522c-59a5-4787-bbfd-89975e5b0954",
    "key": "Org1MSP::x509::CN=org_0,OU=client::CN=fabric_ca.org1.example.com,OU=Hyperledger FireFly,O=org1.example.com,L=Raleigh,ST=North Carolina,C=US",
    "location": {
      "chaincode": "asset_transfer",
      "channel": "firefly"
    },
    "method": {
      "description": "",
      "id": "e5a170d1-0be1-4697-800b-f4bcfaf71cf6",
      "interface": "f1e5522c-59a5-4787-bbfd-89975e5b0954",
      "name": "CreateAsset",
      "namespace": "default",
      "params": [
        {
          "name": "id",
          "schema": {
            "type": "string"
          }
        },
        {
          "name": "color",
          "schema": {
            "type": "string"
          }
        },
        {
          "name": "size",
          "schema": {
            "type": "string"
          }
        },
        {
          "name": "owner",
          "schema": {
            "type": "string"
          }
        },
        {
          "name": "value",
          "schema": {
            "type": "string"
          }
        }
      ],
      "pathname": "CreateAsset",
      "returns": []
    },
    "methodPath": "CreateAsset",
    "type": "invoke"
  },
  "created": "2022-05-02T17:08:40.811630044Z",
  "updated": "2022-05-02T17:08:40.811630044Z"
}
```

You'll notice that we got an ID back with status `Pending`, and that's expected due to the asynchronous programming model of working with custom onchain logic in FireFly. To see what the latest state is now, we can query the chaincode. In a little bit, we'll also subscribe to the events emitted by this chaincode so we can know when the state is updated in realtime.

### Query

To make a read-only request to the blockchain to check the current list of assets, we can make a `POST` to the `query/GetAllAssets` endpoint.

#### Request

`POST` `http://localhost:5000/api/v1/namespaces/default/apis/asset_transfer/query/GetAllAssets`

```json
{}
```

#### Response

```json
[
  {
    "AppraisedValue": 23400,
    "Color": "blue",
    "ID": "asset-01",
    "Owner": "Harry",
    "Size": 30
  }
]
```

> **NOTE:** Some chaincodes may have queries that require input parameters. That's why the query endpoint is a `POST`, rather than a `GET` so that parameters can be passed as JSON in the request body. This particular function does not have any parameters, so we just pass an empty JSON object.
