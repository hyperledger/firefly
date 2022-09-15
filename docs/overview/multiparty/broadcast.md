---
layout: i18n_page
title: pages.broadcast_data
parent: pages.multiparty_features
grand_parent: pages.understanding_firefly
nav_order: 3
---

# Broadcast / shared data
{: .no_toc }

---

## Introduction

Multi-party systems are about establishing a shared source of truth, and
often that needs to include certain reference data that is available
to all parties in the network. The data needs to be "broadcast" to all
members, and also need to be available to new members that join the network

![Multi-party Systems](../../images/multiparty_system1.png "Multi-Party System")

## Blockchain backed broadcast

In order to maintain a complete history of all broadcast data for new members
joining the network, FireFly uses the blockchain to sequence the broadcasts
with pinning transactions referring to the data itself.

Using the blockchain also gives a global order of events for these broadcasts,
which allows them to be processed by each member in a way that allows them
to derive the same result - even though the processing logic on the events
themselves is being performed independently by each member.

For more information see [Multiparty Event Sequencing](../../architecture/multiparty_event_sequencing.html).

## Shared data

The data included in broadcasts is **not** recorded on the blockchain. Instead
a pluggable shared storage mechanism is used to contain the data itself.
The on-chain transaction just contains a hash of the data that is stored off-chain.

This is because the data itself might be too large to be efficiently stored
and transferred via the blockchain itself, or subject to deletion at some
point in the future through agreement by the members in the network.

While the data should be reliably stored with visibility to all members of the
network, the data can still be secured from leakage outside of the network.

The InterPlanetary File System (IPFS) is an example of a distributed technology
for peer-to-peer storage and distribution of such data in a decentralized
multi-party system. It provides secure connectivity between a number of nodes,
combined with a decentralized index of data that is available, and native use
of hashes within the technology as the way to reference data by content.

## FireFly built-in broadcasts

FireFly uses the broadcast mechanism internally to distribute key information to
all parties in the network:

- Network map
  - Organizational identities
  - Nodes
  - See [Identities](../../reference/identities.html) in the reference section for more information
- Datatype definitions
  - See [Datatype](../../reference/types/datatype.html) in the reference section for more information
- Namespaces
  - See [Namespaces](../../reference/namespaces.html) for more information

These definitions rely on the same assurances provided by blockchain backed
broadcast that FireFly applications do.

- Verification of the identity of the party in the network that performed the broadcast
- Deterministic assignment of a namespace+name to an unique item of data
  - If two parties in the network broadcast the same data at similar times, the
    same one "wins" for all parties in the network (including the broadcaster)
