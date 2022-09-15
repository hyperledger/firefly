---
layout: i18n_page
title: pages.multiparty_flow
parent: pages.multiparty_features
grand_parent: pages.understanding_firefly
nav_order: 1
---

# Multiparty Process Flows
{: .no_toc }

---

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

## Building multi-party flows

The ability to globally sequence events _across parties_ is a game changing capability of a multiparty
system. FireFly is designed to allow developers to harnesses that power in the application layer, to build
sophisticated multi-party APIs and user experiences.

[![Multi-party business process flow](../../images/multiparty_business_process_flow.jpg "Multi-party business process flow")](../../images/multiparty_business_process_flow.jpg)

- Build multi-party business processes where there is one agreed outcome:
  - Agree the trigger, inputs, outputs of each step in the process
  - Agree any common "rules of the road" must be adhered to
- Look back at your shared history, when deciding to commit to the next step:
  - Fast rich-query cache, backed by a private database
  - Initiate the next step through automated or manual decision making
  - Only consider a step final once it's multi-party sequence has been confirmed
- Gain big efficiencies in how multi-party business processes work:
  - Once locked in, a step is consider final - attested to by the party
  - If two parties submit conflicting actions, one wins, and one loses
  - Avoids complex compensation logic in the business orchestration layer
  - Provides one clear source of truth to quickly resolve multi-party disputes
- Program multi-party apps using the tools you know:
  - REST APIs for triggering the next step in a process, and querying history
  - WebSockets and Webhooks for events (pluggable to other event transports)
  - Remember - each party runs their own copy of the app, with their own private data
- Allow each party to integrate into their existing core systems:
  - Realtime or batch
  - Human workflows
  - Proprietary business logic that is unique to one party
- Avoid sensitive data written to the blockchain:
  - Works in bi-lateral and multi-lateral scenarios
  - Designed to limit leaking other "metadata" about the transaction as well
  - Share partial history with different participants in a
- No requirement to write custom on-chain smart contract logic:
  - Can be combined with rich custom on-chain logic as well

## Innovate fast

Building a successful multi-party system is often about business experimentation, and business results.
Proving the efficiency gains, and new business models, made possible by working together in a new way
under a new system of trust.

Things that can get in the way of that innovation, can include concerns over data privacy, technology
maturity, and constraints on autonomy of an individual party in the system. An easy to explain position
on how new technology components are used, where data lives, and how business process independence
is maintained can really help parties make the leap of faith necessary to take the step towards a new
model.

Keys to success often include building great user experiences that help digitize clunky decades old
manual processes. Also easy to integrate with APIs, what embrace the existing core systems of record
that are establish within each party.

## Consider the on-chain toolbox too

There is a huge amount of value that deterministic execution of multi-party logic within the blockchain can add.
However, the more compute is made fully deterministic via a blockchain consensus algorithm validated
by multiple parties beyond those with a business need for access to the data, the more sensitivity
needs to be taken to data privacy. Also bear in mind any data that is used in this processing
becomes immutable - it can never be deleted.

The core constructs of blockchain are a great place to start.
Almost every process can be enhanced with pre-built fungible and non-fungible tokens, for example.
Maybe it's to build a token economy that enhances the value parties get from the system,
or to encourage healthy participation (and discourage bad behavior).
Or maybe it's to track exactly which party owns a document, asset, or action within a process using NFTs.

On top of this you can add advanced tools like digital escrow, signature / threshold based voting
on outcomes, and atomic swaps of value/ownership.

The investment in building this bespoke on-chain logic is higher than building the off-chain pieces
(and there are always some off-chain pieces as we've discussed), so it's about finding the kernel
of value the blockchain can provide to differentiate your solution from a centralized database solution.

The power provided by deterministic sequencing of events, attested by signatures, and pinned
to private data might be sufficient for some cases. In others the token constructs are the key value
that differentiates the decentralized ecosystem. Whatever it is, it's important it is identified and
crafted carefully. 

> Note that advanced privacy preserving techniques such as zero-knowledge proofs (ZKP) are gaining traction
> and hardening in their production readiness and efficiency. Expect these to play an increasing
> role in the technology stack of multiparty systems (and Hyperledger FireFly) in the future.

Learn more in the [Deterministic Compute](./deterministic.md) section.
