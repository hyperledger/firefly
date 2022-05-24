FireFly Transactions are a grouping construct for a number of [Operations](./operation.html) and [Events](./event.html)
that need to complete or fail as unit.

> FireFly Transactions are not themselves Blockchain transactions, but in many cases there is
> exactly one Blockchain transaction associated with each FireFly transaction. Exceptions include
> `unpinned` transactions, where there is no blockchain transaction at all.

The Blockchain native transaction ID is stored in the FireFly transaction object when it is known.
However, the FireFly transaction starts before a Blockchain transaction exists - because reliably submitting
the blockchain transaction is one of the operations that is performed inside of the FireFly transaction.

The below screenshot from the FireFly Explorer nicely illustrates how multiple operations and events
are associated with a FireFly transaction. In this example, the transaction tracking is pinning of a batch of
messages stored in IPFS to the blockchain.

So there is a `Blockchain ID` for the transaction - as there is just one Blockchain transaction regardless
of how many messages in the batch. There are operations for the submission of that transaction, and
the upload of the data to IPFS. Then a corresponding `Blockchain Event Received` event for the detection
of the event from the blockchain smart contract when the transaction was mined, and a `Message Confirmed` 
event for each message in the batch (in this case 1). Then here the message was a special `Definition` message
that advertised a new Contract API to all members of the network - so there is a `Contract API Confirmed`
event as well.

[![FireFly Transactions - Explorer View](../../images/firefly_transactions_explorer_view.png)](../../images/firefly_transactions_explorer_view.png)

Each FireFly transaction has a UUID. This UUID is propagated through to all participants in a FireFly transaction.
For example in a [Token Transfer](./tokentransfer.html) that is coordinated with an off-chain private [Message](./message.html),
the transaction ID is propagated to all parties who are part of that transaction. So the same UUID can be used
to find the transaction in the FireFly Explorer of any member who has access to the message.
This is possible because hash-pinned off-chain data is associated with that on-chain transfer.

However, in the case of a raw ERC-20/ERC-721 transfer (without data), or any other raw Blockchain transaction,
the FireFly transaction UUID cannot be propagated - so it will be local on the node that initiated
the transaction.
