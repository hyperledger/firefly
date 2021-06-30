---
layout: default
title: Broadcast / shared data
parent: Key Concepts
nav_order: 7
---

# Broadcast / shared data

---

Multi-party systems are about establishing a shared source of truth, and
often that needs to include certain reference data that is available
to all parties in the network. The data needs to be "broadcast" to all
members, and also need to be available to new members that join the network

![Multi-party Systems](../images/multiparty_system.png "Multi-Party System")

## Blockchain backed broadcast

In order to maintain a complete history of all broadcast data for new members
joining the network, FireFly uses the blockchain as the authoritative source
of truth for broadcast data.

Using the blockchain also gives a global order of events for these broadcasts,
which allows them to be processed by each member in a way that allows them
to derive the same result - even though the processing logic on the events
themselves is being performed independently by each member.
For more information see _Global sequencing_.

## Shared data

The data included in broadcasts is **not** recorded on the blockchain. Instead
a pluggable shared / public storage mechanism is used to contain the data itself.
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
  - See _Identity_ for more information
- Datatype definitions
  - See _Agreed datatypes_ for more information
- Namespaces
  - See _Namespaces_ for more information

## Network Registry

> _Work in progress_

Using a permissioned blockchain and shared data network provides a security mechanism
to protect against broadcast data being published from outside of the network.

However, some networks might have additional permissioning and security requirements
on joining the network. As such FireFly defines a plug-point for a Network Registry
that defines a way to establish authorization to perform a broadcast (that is decoupled
from the blockchain and shared data tiers themselves).
