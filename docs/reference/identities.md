---
layout: default
title: Identities
parent: pages.reference
nav_order: 6
---

# Identities
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

Identities are a critical part of using FireFly in a multi-party system. Every party that joins a multi-party system
must begin by claiming an on- and off-chain _identity_, which is described with a unique [DID](https://www.w3.org/TR/did-core). Each type of
identity is also associated with an on- or off-chain _verifier_, which can be used in some way to check the authorship of a piece
of data. Together, these concepts form the backbone of the trust model for exchanging multi-party data.

## Types of Identities

There are three types of identities:

### org

Organizations are the primary identity type in FireFly. They represent a logical _on-chain_ signing identity, and the
attached verifier is therefore a blockchain key (with the exact format depending on the blockchain being used). Every party
in a multi-party system must claim a _root organization_ identity as the first step to joining the network.

The root organization `name` and `key` must be defined in the [FireFly config](config.html) (once for every multi-party system).
It can be claimed with a POST to `/network/organizations/self`.

Organizations may have child identities of any type.

### node

Nodes represent a logical _off-chain_ identity - and specifically, they are tied to an instance of a data exchange connector.
The format of the attached verifier depends on the data exchange plugin being used, but it will be mapped to some
validation provided by that plugin (ie the name of an X.509 certificate or similar). Every party in a multi-party system must
claim a node identity when joining the network, which must be a child of one of its organization identities (but it is possible
for many nodes to share a parent organization).

The node `name` must be defined in the [FireFly config](config.html) (once for every multi-party system). It can be
claimed with a POST to `/network/nodes/self`.

Nodes must be a child of an organization, and cannot have any child identities of their own.

Note that "nodes" as an identity concept are distinct from FireFly supernodes, from underlying blockchain nodes, and from
anywhere else the term "node" happens to be used.

### custom

Custom identities are similar to organizations, but are provided for applications to define their own more granular notions of
identity. They are associated with an _on-chain_ verifier in the same way as organizations.

They can only have child identities which are also of type "custom".

## Identity Claims

Before an identity can be used within a multi-party system, it must be claimed. The identity claim is a special type of broadcast
message sent by FireFly to establish an identity uniquely among the parties in the multi-party system. As with other broadcasts,
this entails an on-chain transaction which contains a public reference to an off-chain piece of data (such as an IPFS reference)
describing the details of the identity claim.

The claim data consists of information on the identity being claimed - such as the type, the DID, and the parent (if applicable).
The DID must be unique and unclaimed.
The verifier will be inferred from the message - for on-chain identities (org and custom), it is the blockchain key that was used
to sign the on-chain portion of the message, while for off-chain identities (nodes), is is an identifier queried from data exchange.

For on-chain identities with a parent, two messages are actually required - the claim message signed with the new identity's
blockchain key, as well as a separate verification message signed with the parent identity's blockchain key. Both messages must be
received before the identity is confirmed.

## Messaging

In the context of a multi-party system, FireFly provides capabilities for sending off-chain messages that are pinned to
an on-chain proof. The sender of every message must therefore have an on-chain _and_ off-chain identity. For private messages,
every recipient must also have an on-chain and off-chain identity.

### Sender

When sending a message, the on-chain identity of the sender is controlled by the `author` and `key` fields.
* If both are blank, the root organization is assumed.
* If `author` alone is specified, it should be the DID of an org or custom identity. The associated
  verifier will be looked up to use as the `key`.
* If `key` alone is specified, it must match the registered blockchain verifier for an org or custom identity that was previously claimed.
  A reverse lookup will be used to populate the DID for the `author`.
* If `author` and `key` are both specified, they will be used as-is (can be used to send private messages with an unregistered blockchain key).

The resolved `key` will be used to sign the blockchain transaction, which establishes the sender's on-chain identity.

The sender's off-chain identity is always controlled by the `node.name` from the config along with the data exchange plugin.

### Recipients

When specifying private message recipients, each one has an `identity` and a `node`.
* If `identity` alone is specified, it should be the DID of an org or custom identity. The first `node` owned by that identity or one of its
  ancestors will be automatically selected.
* If both `identity` and `node` are specified, they will be used as-is. The `node` should be a child of the given `identity` or one of its
  ancestors.

The `node` in this case will control how the off-chain portion of the message is routed via data exchange.

### Verification

When a message is received, FireFly verifies the following:
* The sender's `author` and `key` are specified in the message. The `author` must be a known org or custom identity. The `key` must match the
  blockchain key that was used to sign the on-chain portion of the message. For broadcast messages, the `key` must match the registered
  verifier for the `author`.
* For private messages, the sending `node` (as reported by data exchange) must be a known node identity which is a child of the message's
  `author` identity or one of its ancestors. The combination of the `author` identity and the `node` must also be found in the message `group`.

In addition, the data exchange plugin is responsible for verifying the sending and receiving identities for the off-chain
data (such as validating the relevant certificates).
