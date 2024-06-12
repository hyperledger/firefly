A Token Transfer is created for each transfer of value that happens under a token pool.

The transfers form an off-chain audit history (an "index") of the transactions that
have been performed on the blockchain.

This historical information cannot be queried directly from the blockchain for most token
implementations, because it is inefficient to use the blockchain to store complex
data structures like this. So the blockchain simply emits events when state changes,
and if you want to be able to query this historical information you need to track
it in your own off-chain database.

Hyperledger FireFly maintains this index automatically for all Token Pools that are configured.

### FireFly initiated vs. non-FireFly initiated transfers

There is no requirement at all to use FireFly to initiate transfers in Token Pools that
Hyperledger FireFly is aware of. FireFly will listen to and update its audit history
and balances for all transfers, regardless of whether they were initiated using a FireFly
Supernode or not.

So you could for example use Metamask to initiate a transfer directly against an ERC-20/ERC-721
contract directly on your blockchain, and you will see it appear as a transfer. Or initiate
a transfer on-chain via another Smart Contract, such as a Hashed Timelock Contract (HTLC) releasing
funds held in digital escrow.

### Message coordinated transfers

One special feature enabled when using FireFly to initiate transfers, is to coordinate an off-chain
data transfer (private or broadcast) with the on-chain transfer of value.  This is a powerful
tool to allow transfers to have rich metadata associated that is too sensitive (or too large)
to include on the blockchain itself.

These transfers have a `message` associated with them, and require a compatible Token Connector and
on-chain Smart Contract that allows a `data` payload to be included as part of the transfer, and to
be emitted as part of the transfer event.

Examples of how to do this are included in the ERC-20, ERC-721 and ERC-1155 Token Connector sample
smart contracts.

### Transfer types

There are three primary types of transfer:

1. Mint - new tokens come into existence, increasing the total supply of tokens
   within the pool. The `from` address will be unset for these transfer types.
2. Burn - existing tokens are taken out of circulation. The `to` address will be
   unset for these transfer types.
3. Transfer - tokens move from ownership by one account, to another account.
   The `from` and `to` addresses are both set for these type of transfers.

Note that the `key` that signed the Transfer transaction might be different to the `from`
account that is the owner of the tokens before the transfer.

The Approval resource is used to track which signing accounts (other than the owner)
have approval to transfer tokens on the owner's behalf.


