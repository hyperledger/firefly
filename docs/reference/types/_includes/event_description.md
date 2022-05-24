Every Event emitted by FireFly shares a common structure.

> See [Events](../events.html) for a reference for how the overall event bus
in Hyperledger FireFly operates, and descriptions of all the sub-categories
of events.

### Sequence

A local `sequence` number is assigned to each event, and you can
use an API to query events using this sequence number in exactly the same
order that they are delivered to your application.

### Reference

Events have a `reference` to the UUID of an object that is the subject of the event,
such as a detailed [Blockchain Event](./(blockchainevent.md), or an off-chain
[Message](./(message.md).

When events are delivered to your application, the `reference` field is
automatically retrieved and included in the JSON payload
that is delivered to your application.

You can use the `?fetchreferences` query parameter on API calls to request the same
in-line JSON payload be included in query results.

The type of the reference also determines what subscription filters apply
when performing server-side filters.

Here is the mapping between event types, and the object that you find in
the `reference` field.

### Correlator

For some event types, there is a secondary reference to an object that is
associated with the event. This is set in a `correlator` field on the 
Event, but is not automatically fetched. This field is primarily used
for the `confirm` option on API calls to allow FireFly to determine
when a request has succeeded/failed.

### Topic

Events have a `topic`, and how that topic is determined is specific to
the type of event. This is intended to be a property you would use to
filter events to your application, or query all historical events
associated with a given business data stream.

For example when you send a [Message](./(message.md), you set the `topics`
you want that message to apply to, and FireFly ensures a consistent global
order between all parties that receive that message.

### Transaction

When actions are submitted by a FireFly node, they are performed
within a FireFly [Transaction](./(transaction.md). The events that occur
as a direct result of that transaction, are tagged with the transaction
ID so that they can be grouped together.

This construct is a distinct higher level construct than a Blockchain
transaction, that groups together a number of operations/events that
might be on-chain or off-chain. In some cases, such as unpinned off-chain
data transfer, a FireFly transaction can exist when there is no
blockchain transaction at all.
Wherever possible you will find that FireFly tags the FireFly transaction
with any associated Blockchain transaction(s).

Note that some events cannot be tagged with a Transaction ID:

- Blockchain events, unless they were part of a batch-pin transaction
  for transfer of a message
- Token transfers/approvals, unless they had a message transfer associated
  with them (and included a `data` payload in the event they emitted)

### Reference, Topic and Correlator by Event Type

| Types                                       | Reference                            | Topic                       | Correlator              |
|---------------------------------------------|--------------------------------------|-----------------------------|-------------------------|
| `transaction_submitted`                     | [Transaction](./(transaction.md)         | `transaction.type`          |                         |
| `message_confirmed`<br/>`message_rejected`  | [Message](./(message.md)                 | `message.header.topics[i]`* | `message.header.cid`    |
| `token_pool_confirmed`                      | [TokenPool](./(tokenpool.md)             | `tokenPool.id`              |                         |
| `token_pool_op_failed`                      | [Operation](./(operation.md)             | `tokenPool.id`              | `tokenPool.id`          |
| `token_transfer_confirmed`                  | [TokenTransfer](./(tokentransfer.md)     | `tokenPool.id`              |                         |
| `token_transfer_op_failed`                  | [Operation](./(operation.md)             | `tokenPool.id`              | `tokenTransfer.localId` |
| `token_approval_confirmed`                  | [TokenApproval](./(tokenapproval.md)     | `tokenPool.id`              |                         |
| `token_approval_op_failed`                  | [Operation](./(operation.md)             | `tokenPool.id`              | `tokenApproval.localId` |
| `namespace_confirmed`                       | [Namespace](./(namespace.md)             | `"ff_definition"`           |                         |
| `datatype_confirmed`                        | [Datatype](./(datatype.md)               | `"ff_definition"`           |                         |
| `identity_confirmed`<br/>`identity_updated` | [Identity](./(identity.md)               | `"ff_definition"`           |                         |
| `contract_interface_confirmed`              | [FFI](./(ffi.md)                         | `"ff_definition"`           |                         |
| `contract_api_confirmed`                    | [ContractAPI](./(contractapi.md)         | `"ff_definition"`           |                         |
| `blockchain_event_received`                 | [BlockchainEvent](./(blockchainevent.md) | From listener **            |                         |
| `blockchain_invoke_op_succeeded`            | [Operation](./(operation.md)             |                             |                         |
| `blockchain_invoke_op_failed`               | [Operation](./(operation.md)             |                             |                         |

> * A separate event is emitted for _each topic_ associated with a [Message](./(message.md).

> ** The topic for a blockchain event is inherited from the blockchain listener,
>    allowing you to create multiple blockchain listeners that all deliver messages
>    to your application on a single FireFly topic.
