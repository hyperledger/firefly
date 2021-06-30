---
layout: default
title: Deterministic compute
parent: Key Concepts
nav_order: 5
---

# Deterministic compute

---

A critical aspect of designing a multi-party systems, is choosing where you exploit
the blockchain and other advanced cryptography technology to automate agreement
between parties.

Specifically where you rely on the computation itself to come up with a result that all parties
can independently trust. For example because all parties performed the same computation
independently and came up with the same result, against the same data, and agreed
to that result using a _consensus algorithm_.

The more sophisticated the agreement is you want to prove, the more consideration needs
to be taken into factors such as:

- Data privacy
- Data deletion
- Ease of understanding by business users
- Ease of audit
- Autonomy of parties with proprietary business logic
- Human workflows (obviously non-deterministic)
- Technology complexity/maturity (particularly for privacy preserving technologies)
- Cost and skills for implementation

FireFly embraces the fact that different use cases, will make different decisions on how much
of the agreement should be enforced through deterministic compute.

Also that multi-party systems include a mixture of approaches in addition to deterministic
compute, including traditional off-chain secure HTTP/Messaging, documents, private
non-deterministic logic, and human workflows.

## The fundamentals

There are some fundamental types of deterministic computation, that can be proved with
mature blockchain technology, and all multi-party systems should consider exploiting:

- Total conservation of value
  - Allows you to assign value to something, because you know it is a fraction of a total pool
  - This is the magic behind fungible tokens, or "coins"
  - The proven technology for this is a shared ledger of all previous transactions
  - _Learn more in the Tokens section_
- Existence and ownership of a unique identifiable thing
  - Gives you an anchor to attach to something in the real world
  - This is the magic behind non-fungible tokens (NTFs)
  - The proven technology for this is a shared ledger of its creation, and ownership changes
  - _Learn more in the Tokens section_
- An agreed sequence of events
  - The foundation tool that allows the building of higher level constructs (including tokens)
  - Not previously available when business ecosystems used HTTP/Messaging transports alone
  - Can be bi-lateral, multi-lateral or global
  - Each blockchain technology has different features to establish these "chains" of information
  - Different approaches provide privacy different levels of privacy on the parties and sequence
- Identification of data by a "hash" of its contents
  - The glue that binds a piece of private data, to a proof that you have a copy of that data
  - This is the basis of "pinning" data to the blockchain, without sharing its contents
  - Care needs to be taken to make sure the data is unique enough to make the hash secure
  - _Learn more in the On-chain/off-chain coordination section_

## Advanced Cryptography and Privacy Preserving Trusted Compute

There are use cases where a deterministic agreement on computation is desired,
but the data upon which the execution is performed cannot be shared between all the parties.

For example proving total conservation of value in a token trading scenario, without
knowing who is involved in the individual transactions. Or providing you have access to a piece of
data, without disclosing what that data is.

Technologies exist that can solve these requirements, with two major categories:

- Zero Knowledge Proofs (ZKPs)
  - Advanced cryptography techniques that allow one party to generate a proof that can be
    be verified by another party, without access to the data used to generate the proof.
- Trusted Compute Environments (TEEs)
  - Secure compute environments that provide proofs of what code was executed, such that
    other parties can be confident of the logic that was executed without having access to the data.

FireFly today provides an orchestration engine that's helpful in coordinating the inputs, outputs,
and execution of such advanced cryptography technologies.

Active collaboration between the FireFly and other projects like Hyperledger Avalon,
and Hyperledger Cactus, is evolving how these technologies can plug-in with higher level patterns.

## Complementary approaches to deterministic computation

Enterprise multi-party systems usually operate differently to end-user decentralized
applications. In particular, strong identity is established for the organizations that are
involved, and those organizations usually sign legally binding commitments around their participation
in the network. Those businesses then bring on-board an ecosystem of employees and or customers that
are end-users to the system.

So the shared source of truth empowered by the blockchain and other cryptography are not the only
tools that can be used in the toolbox to ensure correct behavior. Regognizing that there are real
legal entities involved, that are mature and regulated, does not undermine the value of the blockchain
components. In fact it enhances it.

A multi-party system can use _just enough_ of this secret sauce in the right places, to change
the dynamics of trust such that competitors in a market are willing to create value together that
could never be created before.

Or create a system where parties can share data with each other while still conforming to their own
regulatory and audit commitments, that previously would have been impossible to share.

Not to be overlooked is the sometimes astonishing efficiency increase that can be added to existing
business relationships, by being able to agree the order and sequence of a set of events. Having the
tools to digitize processes that previously took physical documents flying round the world, into
near-immediate digital agreement where the arbitration of a dispute can be resolved at a tiny fraction
of what would have been possible without a shared and immutable audit trail of who said what when.
