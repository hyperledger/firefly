---
layout: i18n_page
title: pages.remote_fabric_network
parent: pages.chains
grand_parent: pages.tutorials
nav_order: 2
has_children: true
---

# Work with remote Hyperledger Fabric Network
{: .no_toc }

The FireFly CLI makes it quick and easy to create an entire FireFly development environment on your local machine, including a new blockchain from scratch. However, sometimes a developer may want to connect their local FireFly development environment to a Fabric network that already exists on their machine or elsewhere. This guide describes the steps to connect FireFly to an external Hyperledger Fabric blockchain network (not created by the FireFly CLI), including interacting with a custom chaincode in order to submit transactions and query for state.

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

For this guide, we will be creating only one supernode for one of the organizations in our fabric network and using that organizational identity to interact with the chaincode. You can create multiple supernodes for different organizations in your fabric network by passing here the number of members in your stack.

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

In `$HOME/.firefly/stacks/my-remote-fabric`, you need to add a msp folder. This msp folder essentially describes the fabric organizational identity which will be interacting with your chaincode. Inside the msp folder, you must have all the certificates and the private key to be used by firefly to query/invoke your chaincode.

You should be having this msp folder for an organizational identity from when you would have set up your fabric network. For more details, please refer to [Hyperledger Fabric Documentation for MSP](https://hyperledger-fabric.readthedocs.io/en/latest/msp.html).

Now, you need to mount this msp directory as a docker volume in your fabconnect service.

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

## Setting up Batchpin chaincode and running your stack

### Deploying batchpin chaincode to fabric network

For Firefly to succeesfully integrate and interact with your remote fabric network, you need to also deploy a Firefly batchpin chaincode in your network setup. This chaincode can be found in the [FireFly git repo](https://github.com/hyperledger/firefly/tree/main/smart_contracts/fabric/firefly-go).

Once you have successfully packaged, installed, approved and commited this chaincode in your remote fabric network, carefully note down the chaincode name you gave and the channel name where this chaincode is deployed for future reference.

> **NOTE:** You can deploy the batchpin chaincode to the same channel where your application chaincode lives or to a separate channel as well.

### Update batchpin chaincode details in firefly core config

In `$HOME/.firefly/stacks/my-remote-fabric/init/config`, you will find a firefly_core_0.yml file which is used by firefly core service in our firefly network setup.

In this file, under the fabconnect section of blockchain-fabric section, you need to change the chaincode value to your batchpin chaincode name, and the channel value to the channel name where your batchpin chaincode was deployed in the previous step.

### Starting your FireFly stack

To start your stack now, simply run:

```
ff start my-remote-fabric
```

This may take a minute or two and in the background the FireFly CLI will do the following for you:

- Download Docker images for all the components of the Supernode
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

## Integrate your FireFly stack with your Fabric chaincode 

Once your stack is up and running, you can define and broadcast FireFly Interface Document for your remote fabric chaincode. You can refer to [Broadcast the Contract Interface(Fabric)](../custom_contracts/fabric.html#broadcast-the-contract-interface) guide for this.

You can also now create an HTTP API for your fabric chaincode which will help you easily query/invoke all your chaincode methods with your organizational context that you set up in fabconnect. You can refer to [Create an HTTP API for the contract(Fabric)](../custom_contracts/fabric.html#create-an-http-api-for-the-contract) guide for this.

To view the OpenAPI spec for your contract, or to submit transactions, query for states and listen for events, you can further refer to [Work with Hyperledger Fabric chaincodes](../custom_contracts/fabric.html#work-with-hyperledger-fabric-chaincodes) guide.
