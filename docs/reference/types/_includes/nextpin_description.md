Next-pins are maintained by each member of a privacy group, in order to detect if a on-chain transaction with a
given "pin" for a message represents the next message for any member of the privacy group.

This allows every member to maintain a global order of transactions within a `topic` in a privacy group, without
leaking the same hash between the messages that are communicated in that group.

> See [Group](./group) for more information on privacy groups.
