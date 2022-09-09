---
layout: i18n_page
title: pages.polygon_testnet
parent: pages.chains
grand_parent: pages.tutorials
nav_order: 1
---


# {%t pages.polygon_testnet %}
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

Starting with FireFly v1.1, it's easy to connect to public Ethereum chains. This guide will walk you through the steps to create a local FireFly development environment and connect it to the public Polygon Mumbai testnet.

## Previous steps: Install the FireFly CLI
If you haven't set up the FireFly CLI already, please go back to the Getting Started guide and read the section on how to [Install the FireFly CLI](../../gettingstarted/firefly_cli.md).

[← ① Install the FireFly CLI](../../gettingstarted/firefly_cli.md){: .btn .btn-purple .mb-5}

## Create an `evmconnect.yml` config file
In order to connect to the Polygon testnet, you will need to set a few configuration options for the evmconnect blockchain connector. Create a text file called `evmconnect.yml` with the following contents:

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
To create a local FireFly development stack and connect it to the Polygon Mumbai testnet, we will use command line flags to customize the following settings:

 - Create a new stack named `polygon` with `1` member
 - Disable `multiparty` mode. We are going to be using this FireFly node as a Web3 gateway, and we don't need to communicate with a consortium here
 - Connect to an ethereum network
 - Use the `evmconnect` blockchain connector
 - Use an remote RPC node. This will create a signer locally, so that our signing key never leaves the development machine.
 - See the polygon [docs](https://docs.polygon.technology/docs/develop/network-details/network/) and select an HTTPS RPC endpoint.
 - Set the chain ID to `80001` (the correct ID for the Polygon Mumbai testnet)
 - Merge the custom config created above with the generated `evmconnect` config file

To do this, run the following command:
```
ff init polygon 1 \
    --multiparty=false \
    -b ethereum \
    -c evmconnect \
    -n remote-rpc \
    --remote-node-url <selected RPC endpoint> \
    --chain-id 80001 \
    --connector-config ~/Desktop/evmconnect.yml 
```

## Start the stack
Now you should be able to start your stack by running:

```
ff start polygon
```

After some time it should print out the following:

```
Web UI for member '0': http://127.0.0.1:5000/ui
Sandbox UI for member '0': http://127.0.0.1:5109


To see logs for your stack run:

ff logs polygon
```

## Get some MATIC
At this point you should have a working FireFly stack, talking to a public chain. However, you won't be able to run any transactions just yet, because you don't have any way to pay for gas. A testnet faucet can give us some MATIC, the native token for Polygon.

First, you will need to know what signing address your FireFly node is using. To check that, you can run:

```
ff accounts list polygon
[
  {
    "address": "0x02d42c32a97c894486afbc7b717edff50c70b292",
    "privateKey": "..."
  }
]
```

Copy the address listed in the output from this command. Go to [https://faucet.polygon.technology/](https://faucet.polygon.technology/) and paste the address in the form. Click the **Submit** button, and then **Confirm**. 

![Polygon Faucet](images/polygon_faucet.png)

### Confirm the transaction on Polygonscan
You should be able to go lookup your account on [Polygonscan for the Mumbai testnet https://mumbai.polygonscan.com/](https://mumbai.polygonscan.com/) and see that you now have a balance of 0.2 MATIC. Simply paste in your account address to search for it.

You can also click on the **Internal Txns** tab from you account page to see the actual transfer of the MATIC from the faucet.

![Polygonscan](images/polygonscan_matic.png)

## Use the public testnet
Now that you have everything set up, you can follow one of the other FireFly guides such as [Using Tokens](../tokens/index.md) or [Custom Smart Contracts](../custom_contracts/ethereum.md). For detailed instructions on deploying a custom smart contract to Polygon, please see the [Polygon docs](https://wiki.polygon.technology/docs/category/deploying-contracts) for instructions using various tools.