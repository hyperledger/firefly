A token approval is a record that an address other than the owner of a token balance,
has been granted authority to transfer tokens on the owners behalf.

The approved "operator" (or "spender") account might be a smart contract,
or another individual.

FireFly provides APIs for initiating and tracking approvals, which token
connectors integrate with the implementation of the underlying token.

The off-chain index maintained in FireFly for allowance allows you to quickly
find the most recent allowance event associated with a pair of keys,
using the `subject` field, combined with the `active` field.
When a new Token Approval event is delivered to FireFly Core by the
Token Connector, any previous approval for the same subject is marked
`"active": false`, and the new approval is marked with `"active": true`

> The token connector is responsible for the format of the `subject` field
> to reflect the owner / operator (spender) relationship.

