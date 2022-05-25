A batch bundles a number of off-chain messages, with associated data, into a single payload
for broadcast or private transfer.

This allows the transfer of many messages (hundreds) to be backed by a single blockchain
transaction. Thus making very efficient use of the blockchain.

The same benefit also applies to the off-chain transport mechanism.

Shared storage operations benefit from the same optimization. In IPFS for example chunks are 256Kb
in size, so there is a great throughput benefit in packaging many small messages into a
single large payload.

For a data exchange transport, there is often cryptography and transport overhead for each individual
transport level send between participants. This is particularly true if using a data exchange
transport with end-to-end payload encryption, using public/private key cryptography for the envelope.

