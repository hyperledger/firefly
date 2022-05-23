A privacy group is a list of identities that should receive a private communication.

When you send a private message, you can specify the list of participants in-line
and it will be resolved to a group. Or you can reference the group using its
identifying hash.

The sender of a message must be included in the group along with the other
participants. The sender receives an event confirming the message, just as
any other participant would do.

> The sender is included automatically in the group when members are
> specified in-line, if it is omitted.

### Group identity hash

The identifying hash for a group is determined as follows:

- All identities are resolved to their DID.
  - An organization name or identity UUID can be used on input
- The UUID of the node that should receive the data for each participant is
  determined (if not specified).
  - The first node found that is in the same identity hierarchy as the
    participant identity, will be chosen.
- The list of participants is ordered by DID, with de-duplication of
  any identities.
- The `namespace`, `name`, and `members` array are then serialized into
  a JSON object, without whitespace.
- A SHA256 hash of the JSON object is calculated

### Private messaging architecture

The mechanism that keeps data private and ordered, without leaking data to the
blockchain, is summarized in the below diagram.

The key points are:

- Data is sent off-chain to all participants via the Data Exchange plugin
  - The Data Exchange is responsible for encryption and off-chain identity verification
- Only parties that are involved in the privacy group receive the data
  - Other parties are only able to view the blockchain transaction
- The hash and member list of the group are not shared outside of the privacy group
  - The `name` of the group can be used as an additional salt in generation of the group hash
  - The member list must be known by all members of the group to verify the blockchain transactions,
    so the full group JSON structure is communicated privately with the first message
    sent on a group
- The blockchain transaction is the source of truth for ordering
  - All members are able to detect a blockchain transaction is part of a group
    they are a member of, from only the blockchain transaction - so they can block
    processing of subsequent messages until the off-chain data arrives (asynchronously)
- The ordering context for messages is masked on the blockchain, so that two messages
  that are for same group do not contain the same context
  - The ordering context (`topic`+`group`) is combined with a `nonce` that is incremented
    for each individual sender, to form a message-specific hash.
  - For each blockchain transaction, this hash can be compared against the expected next
    hash for each member to determine if it is a message on a known group - even without
    the private data (which might arrive later)

[![FireFly Privacy Model](../../images/firefly_data_privacy_model.jpg)](../../images/firefly_data_privacy_model.jpg) 

