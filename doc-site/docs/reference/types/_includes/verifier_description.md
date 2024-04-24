A verifier is a cryptographic verification mechanism for an identity in FireFly.

FireFly generally defers verification of these keys to the lower layers of technologies
in the stack - the blockchain (Fabric, Ethereum etc.) or Data Exchange technology.

As such the details of the public key cryptography scheme are not represented in the
FireFly verifiers. Only the string identifier of the verifier that is appropriate
to the technology.

- Ethereum blockchains: The Ethereum address hex string
- Hyperledger Fabric: The fully qualified MSP Identifier string
- Data exchange: The data exchange "Peer ID", as determined by the DX plugin
