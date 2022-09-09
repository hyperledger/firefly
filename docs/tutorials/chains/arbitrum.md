---
layout: i18n_page
title: pages.arbitrum
parent: pages.chains
grand_parent: pages.tutorials
nav_order: 4
---


# {%t pages.arbitrum %}
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

Starting with FireFly v1.1, it's easy to connect to public Ethereum chains. This guide will walk you through the steps to create a local FireFly development environment and connect it to the Arbitrum Nitro Goerli Rollup Testnet.

## Previous steps: Install the FireFly CLI
If you haven't set up the FireFly CLI already, please go back to the Getting Started guide and read the section on how to [Install the FireFly CLI](../../gettingstarted/firefly_cli.md).

[← ① Install the FireFly CLI](../../gettingstarted/firefly_cli.md){: .btn .btn-purple .mb-5}

## Create an `evmconnect.yml` config file
In order to connect to the Binance Smart Chain testnet, you will need to set a few configuration options for the evmconnect blockchain connector. Create a text file called `evmconnect.yml` with the following contents:

```yml
confirmations:
    required: 4 # choose the number of confirmations you require
policyengine.simple:
    fixedGasPrice: null
    gasOracle:
        mode: connector
```
For more info about `confirmations`, see [Public vs. Permissioned](../../overview/public_vs_permissioned.md)

For this tutorial, we will assume this file is saved at `~/Desktop/evmconnect.yml`. If your path is different, you will need to adjust the path in the next command below.

## Creating a new stack
To create a local FireFly development stack and connect it to the Arbitrum testnet, we will use command line flags to customize the following settings:

 - Create a new stack named `arbitrum` with `1` member
 - Disable `multiparty` mode. We are going to be using this FireFly node as a Web3 gateway, and we don't need to communicate with a consortium here
 - Connect to an ethereum network
 - Use the `evmconnect` blockchain connector
 - Use an remote RPC node. This will create a signer locally, so that our signing key never leaves the development machine.
 - See the Arbitrum [docs](https://developer.offchainlabs.com/node-running/node-providers) and select an HTTPS RPC endpoint.
 - Set the chain ID to `421613` (the correct ID for the Binance Smart Chain testnet)
 - Merge the custom config created above with the generated `evmconnect` config file

To do this, run the following command:
```
ff init arbitrum 1 \
    --multiparty=false \
    -b ethereum \
    -c evmconnect \
    -n remote-rpc \
    --remote-node-url <selected RPC endpoint> \
    --chain-id 421613 \
    --connector-config ~/Desktop/evmconnect.yml
```

## Start the stack
Now you should be able to start your stack by running:

```
ff start arbitrum
```

After some time it should print out the following:

```
Web UI for member '0': http://127.0.0.1:5000/ui
Sandbox UI for member '0': http://127.0.0.1:5109


To see logs for your stack run:

ff logs arbitrum
```

## Get some Aribitrum
At this point you should have a working FireFly stack, talking to a public chain. However, you won't be able to run any transactions just yet, because you don't have any way to pay for gas. 

First, you will need to know what signing address your FireFly node is using. To check that, you can run:

```
ff accounts list arbitrum
[
  {
    "address": "0x225764d1be1f137be23ddfc426b819512b5d0f3e",
    "privateKey": "..."
  }
]
```

Copy the address listed in the output from this command. Next, check out this article [https://medium.com/offchainlabs/new-g%C3%B6rli-testnet-and-getting-rinkeby-ready-for-nitro-3ff590448053](https://medium.com/offchainlabs/new-g%C3%B6rli-testnet-and-getting-rinkeby-ready-for-nitro-3ff590448053) and follow the instructions to send a tweet to the developers. Make sure to change the address to the one in the CLI. 

![Arbitrum Faucet](images/arbitrum_faucet.png)

### Confirm the transaction on Bscscan
You should be able to go lookup your account on [https://goerli-rollup-explorer.arbitrum.io/](https://goerli-rollup-explorer.arbitrum.io/) and see that you now have a balance of 0.001 ether. Simply paste in your account address to search for it.


![Blockscout Scan](images/arbitrum_scan.png)

## Use the public testnet
Now that you have everything set up, you can follow one of the other FireFly guides such as [Using Tokens](../tokens/index.md) or [Custom Smart Contracts](../custom_contracts/ethereum.md). For detailed instructions on deploying a custom smart contract to Binance Smart Chain, please see the [Arbitrum docs](https://developer.offchainlabs.com/docs/Contract_Deployment) for instructions using various tools.