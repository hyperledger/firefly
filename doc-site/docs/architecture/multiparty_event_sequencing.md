---
title: Multiparty Event Sequencing
---

# Multiparty Event Sequencing

<iframe width="736" height="414" src="https://www.youtube.com/embed/bJuu5dMvJ0k" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

[![Multiparty Event Sequencing](../images/global_sequencing.svg "Multiparty Event Sequencing")](../images/global_sequencing.svg)

## Transaction Submission

- An individual FireFly instance preserves the order that it received messages from application instances.
- Where possible, batching is used to roll-up hundreds of transactions into a single blockchain transaction.
- _Blockchain allows these messages to be globally sequenced with messages submitted by other members of the network._

## Blockchain Ordering

- All member FireFly runtimes see every transaction in the same sequence.
- _This includes when transactions are being submitted by both sides concurrently._

## Message Assembly

- A queue of events is maintained for each matching app subscription.
- The public/private payloads travel separately to the blockchain, and arrive at different times. FireFly assembles these together prior to delivery.
- If data associated with a blockchain transaction is late, or does not arrive, all messages on the same "context" will be blocked.
- _It is good practice to send messages that don't need to be processed in order, with different "context" fields. For example use the ID of your business transaction, or other long-running process / customer identifier._

## Event Processing

- Events are processed consistently by all parties.
- All FireFly runtimes see every event that they are subscribed to, in the same sequence.
- _The submitter must also apply the logic only in the sequence ordered by the blockhain. It cannot assume the order even if it is the member that submitted it._
