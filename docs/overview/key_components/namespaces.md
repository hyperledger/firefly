---
layout: default
title: Namespaces
parent: The Key Components
grand_parent: pages.understanding_firefly
nav_order: 5
---

# Apps
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## What is a Supernode

Over the last decade of enterprise blockchain projects, architects and developers have realized
that they need much more than a blockchain node for their projects to be successful.

The development stack needed for an enterprise grade Web3 application,
is just as sophisticated as the stack required for the Web 2.0 applications
that came before.

A raw blockchain node is simply not enough.

## Your project with or without a Supernode

![Without FireFly / with FireFly](../images/without_firefly_with_firefly.png)

So your choice as a development team for a blockchain project becomes whether you build
and update all of the "plumbing" / "middleware" components needed underneath your business
logic yourself, or whether you look for pre-built solutions.

The Hyperledger FireFly approach is to allow the community to collaborate on the development and hardening of
these components, across industries and projects. Then fit them into an open source, enterprise grade,
pluggable development and runtime stack... the _Supernode_.

The application developers then code against these APIs, and can be confident that the business logic that works
on their local laptop against a sandbox, is being written in a way that scales to an enterprise
decentralized application and can be deployed against one or more public/private blockchains in production.

Thus allowing development teams to focus on differentiation where it matters - at the solution layer.

## Types of blockchain project that benefit

There are two main reasons your project might be exploring Supernodes, and considering the open source
approach of Hyperledger FireFly.

### Solution builders

Teams building new decentralized Web3 solutions need a full technology stack for
the application, to manage both private and blockchain data. Particularly in the enterprise space
due to data security, regulatory and privacy concerns.

For theses solutions to be successful, they need decentralized deployment to multiple parties.
Each party needs to customize the deployment to their SecDevOps environment, as well as
onboard it to their key management solution etc.

So the complexity of requiring a bespoke technology stack for a solution can be a barrier to its adoption.

Whereas, building on top of a standardized and open technology stack can ease adoption, as well
as radically reducing the amount of engineering needed by the solution developer.

### Organizations needing a gateway to Web3

Organizations are increasingly participating in multiple blockchain projects, and integrating with
digital assets in multiple blockchain ecosystems.

This means core IT security policy needs to scale to the challenge of adding these connections,
and managing the wallets / signing identities, data flow, and SecDevOps requirements across multiple
projects.

A gateway tier at the edge between the core systems of the enterprise, and the Web3 transactions,
helps reduce the overhead, and reduce risk.

## Feature view

So what makes a Supernode?

![Hyperledger FireFly features](../images/firefly_functionality_overview.png)

Let's break down the functionality that is needed for an enterprise blockchain solution.

## Application features

Rapidly accelerating development is a key requirement of any Supernode.

The business logic APIs, web and mobile user experiences for Web3 applications need to be just as rich
and feature-full as the Web 2.0 / centralized applications.

That means developers skilled in these application layers, must have the tools they need.

Capabilities fitting their application development toolchain, and optimized to their skillset.

### API Gateway

Modern APIs that:

- Are fast and efficient
- Have rich query support
- Give deterministic outcomes and clear instruction for safe use
- Integrate with your security frameworks like OAuth 2.0 / OpenID Connect single sign-on
- Provide Open API 3 / Swagger definitions
- Come with code SDKs, with rich type information
- Conform as closely as possible to the principles of REST
- Do not pretend to be RESTful in cases when it is impossible to be

### Event Streams

The reality is that the only programming paradigm that works for a decentralized solutions,
is an event-driven one.

All blockchain technologies are for this reason event-driven programming interfaces at their core.

In an overall solution, those on-chain events must be coordinated with off-chain private
data transfers, and existing core-systems / human workflows.

This means great event support is a must:

- Convenient WebSocket APIs that work for your microservices development stack
- Support for Webhooks to integrated serverless functions
- Integration with your core enterprise message queue (MQ) or enterprise service bus (ESB)
- At-least-once delivery assurance, with simple instructions at the application layer

### API Generation

The blockchain is going to be at the heart of your Web3 project. While usually small in overall surface
area compared to the lines of code in the traditional application tiers, this kernel of
mission-critical code is what makes your solution transformational compared to a centralized / Web 2.0 solution.

Whether the smart contract is hand crafted for your project, an existing contract on a public blockchain,
or a built-in pattern of a framework like FireFly - it must be interacted with correctly.

So there can be no room for misinterpretation in the hand-off between the blockchain
Smart Contract specialist, familiar with EVM contracts in Solidity/Vyper, Fabric chaincode
(or maybe even raw block transition logic in Rust or Go), and the backend/full-stack
application developer / core-system integrator.

Well documented APIs are the modern norm for this, and it is no different for blockchain. This means:

- Generating the interface for methods and events on your smart contract
- Providing robust transaction submission, and event streaming
- Publishing the API, version, and location, of your smart contracts to the network

## Flow features

Data, value, and process flow are how decentralized systems function. In an enterprise context
not all of this data can be shared with all parties, and some is very sensitive.

### Private data flow

Managing the flows of data so that the right information is shared with the right parties,
at the right time, means thinking carefully about what data flows over what channel.

The number of enterprise solutions where all data can flow directly through the blockchain,
is vanishingly small.

Coordinating these different data flows is often one of the biggest pieces of heavy lifting solved
on behalf of the application by a robust framework like FireFly:

- Establishing the identity of participants so data can be shared
- Securing the transfer of data off-chain
- Coordinating off-chain data flow with on-chain data flow
- Managing sequence for deterministic outcomes for all parties
- Integrating off-chain private execution with multi-step stateful business logic

### Multi-party business process flow

Web3 has the potential to transform how ecosystems interact. Digitally transforming
legacy process flows, by giving deterministic outcomes that are trusted by all parties,
backed by new forms of digital trust between parties.

Some of the most interesting use cases require complex multi-step business process across
participants. The Web3 version of business process management, comes with a some new challenges.

So you need the platform to:

- Provide a robust event-driven programming model fitting a "state machine" approach
- Integrate with the decentralized application stack of each participant
- Allow integration with the core-systems and human decision making of each participant
- Provide deterministic ordering between all parties
- Provide identity assurance and proofs for data flow / transition logic

### Data exchange

Business processes need data, and that data comes in many shapes and sizes.

The platform needs to handle all of them:

- Large files and documents, as well as application data
- Uniqueness / Enterprise NFTs - agreement on a single "foreign key" for a record
- Non-repudiation, and acknowledgement of receipt
- Coordination of flows of data, with flows of value - delivery vs. payment scenarios

## Digital asset features

The modelling, transfer and management of digital assets is the core programming
foundation of blockchain.

Yet out of the box, raw blockchains designed to efficiently manage these assets
in large ecosystems, do not come with all the building blocks needed by applications.

### Token API

Tokens are such a fundamental construct, that they justify a standard API.
This has been evolving in the industry through standards like ERC-20/ERC-721,
and Web3 signing wallets and that support these.

Supernodes bring this same standardization to applications. Providing APIs
that work across token standards, and blockchain implementations, providing
consistent and interoperable support.

This means one application or set of back-end systems, can integrate with multiple
blockchains, and different token implementations.

Pluggability here is key, so that the rules of governance of each digital
asset ecosystem can be exposed and enforced. Whether tokens are fungible,
non-fungible, or some hybrid in between.

### Transfer history / audit trail

For efficiency blockchains seldom provide in their core the ability to
query historical transaction information. Sometimes even the ability
to query balances is unavailable, for blockchains based on a UTXO model.

So off-chain indexing of transaction history is an absolute must-have
for any digital asset solution, or even a simple wallet application.

A platform like Hyperledger FireFly provides:

- Automatic indexing of tokens, whether existing or newly deployed
- Off-chain indexing of fungible and non-fungible asset transfers & balances
- Off-chain indexing of approvals
- Integration with digital identity
- Full extensibility across both token standards and blockchain technologies

### Wallets

Wallet and signing-key management is a critical requirement for any
blockchain solution, particularly those involving the transfer
of digital assets between wallets.

A platform like Hyperledger FireFly provides you the ability to:

- Integrate multiple different signing/custody solutions in a proven way
- Manage the mapping of off-chain identities to on-chain signing identities
- Provide a plug-point for policy-based decision making on high value transactions
- Manage connections to multiple different blockchain solutions
