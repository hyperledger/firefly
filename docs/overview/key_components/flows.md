---
layout: default
title: Flows
parent: The Key Components
grand_parent: pages.understanding_firefly
nav_order: 4
---

# Flows
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

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