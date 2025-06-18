---
title: Cardano
---

This guide will walk you through the steps to create a local FireFly development environment running against the preview node.

## Previous steps: Install the FireFly CLI

If you haven't set up the FireFly CLI already, please go back to the Getting Started guide and read the section on how to [Install the FireFly CLI](../../gettingstarted/firefly_cli.md).

[← ① Install the FireFly CLI](../../gettingstarted/firefly_cli.md){: .md-button .md-button--primary}

## Create the stack

A Cardano stack can be run in two different ways; using a local Cardano node, or a remote Blockfrost address.

### Option 1: Use a local Cardano node

> **NOTE**: The cardano-node communicates over a Unix socket, so this will not work on Windows.

Start a local Cardano node. The fastest way to do this is to [use mithril](https://mithril.network/doc/manual/getting-started/bootstrap-cardano-node/) to bootstrap the node.

For an example of how to bootstrap and run the Cardano node in Docker, see [the firefly-cardano repo](https://github.com/hyperledger/firefly-cardano/blob/1be3b08d301d6d6eeb5b79e40cf3dbf66181c3de/infra/docker-compose.node.yaml#L4).

The cardano-node exposes a Unix socket named `node.socket`. Pass that to firefly-cli. The example below uses `firefly-cli` to
 - Create a new Cardano-based stack named `dev`.
 - Connect to the local Cardano node, which is running in the [preview network](https://preview.cexplorer.io/).

```sh
ff init cardano dev \
    --network preview \
    --socket /path/to/ipc/node.socket
```

### Option 2: Use Blockfrost

The Cardano connector can also use the [paid Blockfrost API](https://blockfrost.io/) in place of a local Cardano node.

The example below uses firefly-cli to
 - Create a new Cardano-based stack named `dev`
 - Use the given blockfrost key for the preview network.

```sh
ff init cardano dev \
    --network preview \
    --blockfrost-key previewXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

## Start the stack

Now you should be able to start your stack by running:

```sh
ff start dev
```

After some time it should print out the following:

```
Web UI for member '0': http://127.0.0.1:5000/ui
Sandbox UI for member '0': http://127.0.0.1:5109


To see logs for your stack run:

ff logs dev
```

## Get some ADA

Now that you have a stack, you need some seed funds to get started. Your stack was created with a wallet already (these are free to create in Cardano). To get the address, you can run
```sh
ff accounts list dev
```

The response will look like
```json
[
  {
    "address": "addr_test1...",
    "privateKey": "..."
  }
]
```

If you're developing against a testnet such as preview, you can receive funds from the [testnet faucet](https://docs.cardano.org/cardano-testnets/tools/faucet). Pass the `address` from that response to the faucet.
