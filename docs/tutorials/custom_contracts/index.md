---
layout: i18n_page
title: pages.custom_smart_contracts
parent: pages.tutorials
nav_order: 7
has_children: true
---

## Quick reference

Almost all blockchain platforms offer the ability to execute smart contracts on-chain in order to manage states on the shared ledger. FireFly provides support to use RESTful APIs to interact with the smart contracts deployed in the target blockchains, and listening to events via websocket.

FireFly's unified API creates a consistent application experience regardless of the specific underlying blockchain implementation. It also provides developer-friendly features like automatic OpenAPI Specification generation for smart contracts, plus a built-in Swagger UI.

## Key concepts

FireFly defines the following constructs to support custom smart contracts:

- **Contract Interface**: FireFly defines a common, blockchain agnostic way to describe smart contracts. This is referred to as a Contract Interface. A contract interface is written in the FireFly Interface (FFI) format. It is a simple JSON document that has a name, a namespace, a version, a list of methods, and a list of events.

For more details, you can also have a look at the [Reference page for the FireFly Interface Format](../../reference/firefly_interface_format).

For blockchains that offer a DSL describing the smart contract interface, such as Ethereum's ABI (Application Binary Interface), FireFly offers a convenience tool to convert the DSL into the FFI format.

> **NOTE**: Contract interfaces are scoped to a namespace. Within a namespace each contract interface must have a unique name and version combination. The same name and version combination can exist in _different_ namespaces simultaneously.

- **HTTP API**: Based on a Contract Interface, FireFly further defines an HTTP API for the smart contract, which is complete with an OpenAPI Specification and the Swagger UI. An HTTP API defines an `/invoke` root path to submit transactions, and a `/query` root path to send query requests to read the state back out.

How the invoke vs. query requests get interpreted into the native blockchain requests are specific to the blockchain's connector. For instance, the Ethereum connector translates `/invoke` calls to `eth_sendTransaction` JSON-RPC requests, while `/query` calls are translated into `eth_call` JSON-RPC requests. One the other hand, the Fabric connector translates `/invoke` calls to the multiple requests required to submit a transaction to a Fabric channel (which first collects endorsements from peer nodes, and then sends the assembled transaction payload to an orderer, for details please refer to the Fabric documentation).

- **Blockchain Event Listener**: Regardless of a blockchain's specific design, transaction processing are always asynchronous. This means a transaction is submitted to the network, at which point the submitting client gets an acknowledgement that it has been accepted for further processing. The client then listens for notifications by the blockchain when the transaction gets committed to the blockchain's ledger.

FireFly defines event listeners to allow the client application to specify the relevant blockchain events to keep track of. A client application can then receive the notifications from FireFly via an event subscription.

- **Event Subscription**: While an event listener tells FireFly to keep track of certain events emitted by the blockchain, an event subscription tells FireFly to relay those events to the client application. Each subscriptions represents a stream of events that can be delivered to a listening client with various modes of delivery with at-least-once delivery guarantee.

This is exactly the same as listening for any other events from FireFly. For more details on how Subscriptions work in FireFly you can read the [Getting Started guide to Listen for events](../events.md).

## Custom onchain logic async programming in FireFly

Like the rest of FireFly, custom onchin logic support are implemented with an asynchronous programming model. The key concepts here are:

- Transactions are submitted to FireFly and an ID is returned. This is the **Operation ID**.
- The transaction itself happens asynchronously from the HTTP request that initiated it
- Blockchain events emitted by the custom onchain logic (Ethereum smart contracts, Fabric chaincodes, Corda flows, etc.) will be stored in FireFly's database if FireFly has a **Event Listener** set up for that specific type of event. FireFly will also emit an event of type `blockchain_event_received` when this happens.

<!-- TODO: Update this diagram -->

![Smart Contracts Async Flow](../../images/smart_contracts_async_flow.svg "Smart Contracts Async Flow")
