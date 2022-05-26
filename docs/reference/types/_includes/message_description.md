Message is the envelope by which coordinated data exchange can happen between parties in the network. Data is passed by reference in these messages, and a chain of hashes covering the data and the details of the message, provides a verification against tampering.

A message is made up of three sections:

1. The header - a set of metadata that determines how the message is ordered, who should receive it, and how they should process it 
2. The data - an array of data attachments
3. Status information - fields that are calculated independently by each node, and hence update as the message makes it way through the system

### Hash

Sections (1) and (2) are fixed once the message is sent, and a `hash` is generated that provides tamper protection.

The hash is a function of the header, and all of the data payloads. Calculated as follows:

- The hash of each Data element is calculated individually
- A JSON array of `[{"id":"{{DATA_UUID}}","hash":"{{DATA_HASH}}"}]` is hashed, and that hash is stored in `header.datahash`
- The `header` is serialized as JSON with the deterministic order (listed [below](#messageheader)) and hashed
  - JSON data is serialized without whitespace to hash it.
  - The hashing algorithm is SHA-256

Each node independently calculates the hash, and the hash is included in the manifest of the [Batch](./batch.html) by the
node that sends the message.
Because the hash of that batch manifest is included in the blockchain transaction, a message transferred to
a node that does not match the original message hash is rejected.

### Tag

The `header.tag` tells the processors of the message how it should be processed, and  what data they should expect it to contain.

If you think of your decentralized application like a state machine, then you need to have a set of well defined transitions
that can be performed between states. Each of these transitions that requires off-chain transfer of private data
(optionally coordinated with an on-chain transaction) should be expressed as a type of message, with a particular `tag`.

Every copy of the application that runs in the participants of the network should look at this `tag` to determine what
logic to execute against it.

> Note: For consistency in ordering, the sender should also wait to process the state machine transitions associated
> with the message they send until it is ordered by the blockchain. They should not consider themselves special because
> they sent the message, and process it immediately - otherwise they could end up processing it in a different order
> to other parties in the network that are also processing the message.

### Topics

The `header.topics` strings allow you to set the the _ordering context_ for each message you send, and you are strongly
encouraged to set it explicitly on every message you send (falling back to the `default` topic is not recommended).

A key difference between blockchain backed decentralized applications and other event-driven applications, is
that there is a single source of truth for the order in which things happen.

In a multi-party system with off-chain transfer of data as well as on-chain transfer of data, the two sets of
data need to be coordinated together. The off-chain transfer might happen at different times, and is subject to the reliability
of the parties & network links involved in that off-chain communication. 

A "stop the world" approach to handling a single piece of missing data is not practical for a high volume
production business network.

The ordering context is a function of:

1. Whether the message is broadcast or private
2. If it is private, the privacy group associated with the message
3. The `topic` of the message

When an on-chain transaction is detected by FireFly, it can determine the above ordering - noting that privacy is preserved
for private messages by masking this ordering context message-by-message with a nonce and the group ID, so that only the
participants in that group can decode the ordering context.

If a piece of off-chain data is unavailable, then the FireFly node will block only streams of data that are associated
with that ordering context.

For your application, you should choose the most granular identifier you can for your `topic` to minimize the scope
of any blockage if one item of off-chain data fails to be delivered or is delayed. Some good examples are:

- A business transaction identifier - to ensure all data related to particular business transaction are processed in order
- A globally agreed customer identifier - to ensure all data related to a particular business entity are processed in order

#### Using multiple topics

There are some advanced scenarios where you need to _merge streams_ of ordered data, so that two previously separately
ordered streams of communication (different state machines) are joined together to process a critical decision/transition
in a deterministic order.

A _synchronization point_ between two otherwise independent streams of communication.

To do this, simply specify two `topics` in the message you sent, and the message will be independently ordered against
both of those topics.

> You will also receive **two events** for the confirmation of that message, one for each topic.

Some examples:

- Agreeing to join two previously separate business transactions with ids `000001` and `000002`, by discarding business transaction `000001` as a duplicate
  - Specify `topics: ["000001","000002"]` on the special merge message, and then from that point onwards you would only need to specify `topics: ["000002"]`.
- Agreeing to join two previously separate entities with `id1` and `id2`, into a merged entity with `id3`. 
  - Specify `topics: ["id1","id2","id3"]` on the special merge message, and then from that point onwards you would only need to specify `topics: ["id3"]`.

### Transaction type

By default messages are pinned to the blockchain, within a [Batch](./batch.html).

For private messages, you can choose to disable this pinning by setting `header.txtype: "unpinned"`.

Broadcast messages must be pinned to the blockchain.

### In-line data

When sending a message you can specify the array of [Data](./data.html) attachments in-line, as part of the same JSON payload.

For example, a minimal broadcast message could be:

```js
{
    "data": [
        {"value": "hello world"}
    ]
}
```

When you send this message with `/api/v1/namespaces/{ns}/messages/broadcast`:
- The `header` will be initialized with the default values, including `txtype: "batch_pin"`
- The `data[0]` entry will be stored as a Data resource
- The message will be assembled into a batch and broadcast
