Blockchain Events are detected by the blockchain plugin:

1. When a [ContractListener](./contractlistener.html) has been
   configured against any custom smart contract through the FireFly API
2. Indirectly via a Token Connector, which understands the correct events
   to listen to for a [Token Pool](./tokenpool.html) configured against a
   standard such as ERC-20/ERC-721/ERC-1155
3. Automatically by FireFly core, for the BatchPin contract that can
   be used for high throughput batched pinning of off-chain data transfers
   to the blockchain (complementary to using your own smart contracts).

### Protocol ID

Each Blockchain Event (once final) exists in an absolute location somewhere
in the transaction history of the blockchain. A particular slot, in a particular
block.

How to describe that position contains blockchain specifics - depending on how
a particular blockchain represents transactions, blocks and events (or "logs").

So FireFly is flexible with a string `protocolId` in the core object to
represent this location, and then there is a convention that is adopted by
the blockchain plugins to try and create some consistency.

An example `protocolId` string is: `000000000041/000020/000003`

- `000000000041` - this is the block number
- `000020` - this is the transaction index within that block
- `000003` - this is the event (/log) index within that transaction


The string is alphanumerically sortable as a plain string;

> Sufficient zero padding is included at each layer to support future expansion
> without creating a string that would no longer sort correctly.



