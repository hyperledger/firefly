---
layout: i18n_page
title: pages.zksync_testnet
parent: pages.chains
grand_parent: pages.tutorials
nav_order: 11
---


# {%t pages.zksync_testnet %}
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

Starting with FireFly v1.1, it's easy to connect to public Ethereum chains. This guide will walk you through the steps to create a local FireFly development environment and connect it to the zkSync testnet.

## Previous steps: Install the FireFly CLI
If you haven't set up the FireFly CLI already, please go back to the Getting Started guide and read the section on how to [Install the FireFly CLI](../../gettingstarted/firefly_cli.md).

[← ① Install the FireFly CLI](../../gettingstarted/firefly_cli.md){: .btn .btn-purple .mb-5}

## Create an `evmconnect.yml` config file
In order to connect to the zkSync testnet, you will need to set a few configuration options for the evmconnect blockchain connector. Create a text file called `evmconnect.yml` with the following contents:

```yml
confirmations:
    required: 4 # choose the number of confirmations you require
policyengine.simple:
    fixedGasPrice: null
    gasOracle:
        mode: connector
```
For this tutorial, we will assume this file is saved at `~/Desktop/evmconnect.yml`. If your path is different, you will need to adjust the path in the next command below.

## Creating a new stack
To create a local FireFly development stack and connect it to the zkSync testnet, we will use command line flags to customize the following settings:

 - Create a new stack named `zkSync` with `1` member
 - Disable `multiparty` mode. We are going to be using this FireFly node as a Web3 gateway, and we don't need to communicate with a consortium here
 - Connect to an ethereum network
 - Use the `evmconnect` blockchain connector
 - Use an remote RPC node. This will create a signer locally, so that our signing key never leaves the development machine.
 - See the list of providers for zkSync [docs](https://v2-docs.zksync.io/dev/fundamentals/testnet.html). For this tutorial we will use `https://zksync2-testnet.zksync.dev`
 - Set the chain ID to `280` (the correct ID for the zkSync testnet)
 - Merge the custom config created above with the generated `evmconnect` config file

To do this, run the following command:
```
ff init zksync 1\
    --multiparty=false \
    -b ethereum \
    -c evmconnect \
    -n remote-rpc \
    --remote-node-url https://zksync2-testnet.zksync.dev\
    --chain-id 280 \
    --connector-config ~/Desktop/evmconnect.yml
```

## Start the stack
Now you should be able to start your stack by running:

```
ff start zksync
```

After some time it should print out the following:

```
Web UI for member '0': http://127.0.0.1:5000/ui
Sandbox UI for member '0': http://127.0.0.1:5109


To see logs for your stack run:

ff logs zksync
```

## Get some ETH
At this point you should have a working FireFly stack, talking to a public chain. However, you won't be able to run any transactions just yet, because you don't have any way to pay for gas. zkSync does not currently have its own native token and instead uses Ethereum for transaction. A testnet faucet can give us some ETH.

First, you will need to know what signing address your FireFly node is using. To check that, you can run:

```
ff accounts list zkSync
[
  {
    "address": "0x8cf4fd38b2d56a905113d23b5a7131f0269d8611",
    "privateKey": "..."
  }
]
```

Copy your zkSync address and go to the [Goerli Ethereum faucet](https://www.allthatnode.com/faucet/ethereum.dsrv) and paste the address in the form. Click the **Request Tokens** button. Note that any Goerli Ethereum faucet will work.

![Goerli Faucet](images/zksync_faucet.png)

### Confirm the transaction on the Etherscan Explorer
You should be able to go lookup your account at [https://etherscan.io/](https://etherscan.io/) and see that you now have a balance of 0.025 ETH. Simply paste in your account address to search for it.

![Etherscan](images/zksync_explorer.png)

## Use the public testnet
Now that you have everything set up, you can follow one of the other FireFly guides such as [Using Tokens](../tokens/index.md) or [Custom Smart Contracts](../custom_contracts/ethereum.md). For detailed instructions on deploying a custom smart contract to zkSync, please see the [zkSync docs](https://docs.zksync.io/dev/contracts/) for instructions using various tools.