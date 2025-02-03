---
title: Undelivered messages
---

When using FireFly in multiparty mode to deliver broadcast or private messages, one potential problem is that of
undelivered messages. In general FireFly's message delivery service should be extremely reliable, but understanding
when something has gone wrong (and how to recover) can be important for maintaining system health.

## Background

This guide assumes some familiarity with how
[multiparty event sequencing](../architecture/multiparty_event_sequencing.md) works.
In general, FireFly messages come in three varieties:

![FireFly Message Types](../images/firefly_message_types.png "FireFly Message Types")

1. **Unpinned private messages:** private messages delivered directly via data exchange
2. **Pinned private messages:** private messages delivered via data exchange, with a hash of the message recorded on the blockchain ledger
3. **Pinned broadcast messages:** messages stored in IPFS, with a hash and reference to the message shared

All messages are batched for efficiency, but in cases of low throughput, you may frequently see batches
containing exactly one message.

"Pinned" messages are those that use the blockchain ledger for reliable timestamping and ordering. These messages have
two pieces which must be received before the message can be processed: the **batch** is the actual contents of
the message(s), and the **pin** is the lightweight blockchain transaction that records the existence and ordering of
that batch. We frequently refer to this combination as a **batch-pin**.

> Note: there is a fourth type of message denoted with the type "definition", used for things such as identitity claims
> and advertisement of contract APIs. For most troubleshooting purposes these can be treated the same as pinned
> broadcast messages, as they follow the same pattern (with only a few additional processings steps inside FireFly).

## Symptoms

When some part of the multiparty messaging infrastructure requires troubleshooting, common symptoms include:

- a message was sent, but is not present on some other node where it should have been received
- a message is stuck indefinitely in "sent" or "pending" state

## Troubleshooting steps

When troubleshooting one of the symptoms above, the main goal is to identify the specific piece of the infrastructure that is
experiencing an issue. This can lead you to diagnose specific issues such as misconfiguration, network problems, database
integrity problems, or potential code bugs.

In all cases, the **batch ID** is the most critical piece of data for determining the nature of the issue. You can usually
retrieve the batch for a particular message by querying `/messages/<message-id>` and looking for the `batch` field in the returned
response. In rare cases, if this is not populated, you can also retrieve the message transaction via `/messages/<message-id>/transaction`,
and then you can use the transaction ID to query `/batches?tx.id=<transaction-id>`.

The batch ID will be the same on all nodes involved in the messaging flow. Therefore, the following two steps can be
easily performed to check for the existence of the expected items:

- query `/batches/<batch-id>` on each node that should have the message
- query `/pins?batch=<batch-id>` on each node that should have the message (for pinned messages only)

Then choose one of these scenarios to focus in on an area of interest:

#### 1) Is the batch missing on a node that should have received it?

For private messages, this indicates a potential problem with **data exchange**. Check the sending node to see if the FireFly
operations succeeded when sending the batch via data exchange, and check the data exchange logs for any issues processing it
(the FireFly operation ID can be used to trace the operation through data exchange as well).
If an operation failed on the sending node, you may need to retry it with `/operations/<op-id>/retry`.

For broadcast messages, this indicates a potential problem with **IPFS**. Check the sending node to see if the FireFly
operations succeeded when uploading the batch to IPFS, and the receiving node to see if the operations succeeded when
downloading the batch from IPFS. If an operation failed, you may need to retry it with `/operations/<op-id>/retry`.

#### 2) Is the batch present, but the pin is missing?

This indicates a potential problem with the **blockchain connector**. Check if the underlying blockchain node is
healthy and mining blocks. Check the sending FireFly node to see if the operation succeeded when pinning the batch via the
blockchain. Check the blockchain connector logs (such as evmconnect or fabconnect) to see if it is
successfully processing events from the blockchain, or if it is encountering any errors before forwarding those events
on to FireFly.

#### 3) Are the batch and pin both present, but the messages from the batch are still stuck in "sent" or "pending"?

Check the pin details to see if it contains a field `"dispatched": true`. If this field is false or missing, it means
that the pin was received but couldn't be matched successfully with the off-chain batch contents. Check the FireFly
logs and search for the batch ID - likely this issue is in FireFly and it will have logged some problem while
aggregating the batch-pin. In some cases, the FireFly logs may indicate that the pin could not be dispatched because
it was "stuck" behind another pin on the same context - so you may need to follow the trail to a batch-pin for a
different batch and determine why that earlier one was not processed (by starting over on this rubric
and troubleshooting that batch).

## Opening an issue

It's possible that the above steps may lead to an obvious solution (such as recovering a crashed service or retrying a
failed operation). If they do not, you can open an issue. The more detail you can include from the troubleshooting above
(including the type of message, the nodes involved, and the details on the batch and pin found when examining each node),
the more likely it is that someone can help to suggest additional troubleshooting. Full logs from FireFly, and (as
deemed relevant from the troubleshooting above) full logs from the data exchange or blockchain connector runtimes, will
also make it easier to offer additional insight.
