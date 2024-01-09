---
layout: i18n_page
title: pages.tezos_testnet
parent: pages.chains
grand_parent: pages.tutorials
nav_order: 6
---

# {%t pages.tezos_testnet %}
{: .no_toc }

1. TOC
{:toc}

---

This guide will walk you through the steps to create a local FireFly development environment and connect it to the public Tezos Ghostnet testnet.

## Previous steps: Install the FireFly CLI

If you haven't set up the FireFly CLI already, please go back to the Getting Started guide and read the section on how to [Install the FireFly CLI](../../gettingstarted/firefly_cli.md).

[← ① Install the FireFly CLI](../../gettingstarted/firefly_cli.md){: .btn .btn-purple .mb-5}

## Set up the transaction signing service <a name="signatory"></a>

[Signatory](https://signatory.io/) service allows to work with many different key-management systems.\
By default, FF uses [local signing](https://signatory.io/docs/file_based) option.\
However, it is also possible to configure the transaction signing service using key management systems such as: AWS/Google/Azure KMS, HCP Vault, etc.
> **NOTE**: The default option is not secure and is mainly used for development and demo purposes. Therefore, for the production, use the selected KMS.\
The full list can be found [here](https://github.com/ecadlabs/signatory#backend-kmshsm-support-status).

## Creating a new stack

To create a local FireFly development stack and connect it to the Tezos Ghostnet testnet, we will use command line flags to customize the following settings:

- Create a new Tezos based stack named `tezos` with `1` member
- Disable `multiparty` mode. We are going to be using this FireFly node as a Web3 gateway, and we don't need to communicate with a consortium here
- See the list of Tezos [public RPC nodes](https://tezostaquito.io/docs/rpc_nodes/) and select an HTTPS RPC node.

To do this, run the following command:

```
ff init tezos dev 1 \
    --multiparty=false \
    --remote-node-url <selected RPC endpoint>
```

> **NOTE**: The public RPC nodes may have limitations or may not support all FF required RPC endpoints. Therefore it's not recommended to use ones for production and you may need to run own node or use third-party vendors.

## Start the stack

Now you should be able to start your stack by running:

```
ff start dev
```

After some time it should print out the following:

```
Web UI for member '0': http://127.0.0.1:5000/ui
Sandbox UI for member '0': http://127.0.0.1:5109


To see logs for your stack run:

ff logs dev
```

## Get some XTZ

At this point you should have a working FireFly stack, talking to a public chain. However, you won't be able to run any transactions just yet, because you don't have any way to pay transaction fee. A testnet faucet can give us some XTZ, the native token for Tezos.

First, you need to get an account address, which was created during [signer set up](#signatory) step.\
To check that, you can run:
```
ff accounts list dev
[
  {
    "address": "tz1cuFw1E2Mn2bVS8q8d7QoCb6FXC18JivSp",
    "privateKey": "..."
  }
]
```


After that, go to [Tezos Ghostnet Faucet](https://faucet.ghostnet.teztnets.xyz/) and paste the address in the form and click the **Request** button.

![Tezos Faucet](images/tezos_faucet.png)

### Confirm the transaction on TzStats
You should be able to go lookup your account on [TzStats for the Ghostnet testnet](https://ghost.tzstats.com/) and see that you now have a balance of 100 XTZ (or 2001 XTZ accordingly). Simply paste in your account address to search for it.

On the **Transfers** tab from you account page you will see the actual transfer of the XTZ from the faucet.

![TzStats](images/tezos_explorer.png)

## Use the public testnet

Now that you have everything set up, you can follow one of the other FireFly guides such as [Custom Smart Contracts](../custom_contracts/tezos.md). For detailed instructions on deploying a custom smart contract to Tezos, please see the [Tezos docs](https://docs.tezos.com/smart-contracts/deploying) for instructions using various tools.