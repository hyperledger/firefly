Contract APIs provide generated REST APIs for on-chain smart contracts.

API endpoints are generated to invoke or perform query operations against
each of the functions/methods implemented by the smart contract.

API endpoints are also provided to add listeners to the events of that
smart contract.

> Note that once you have established listeners for your blockchain events
> into FireFly, you need to also subscribe in your application to receive
> the FireFly events (of type `blockchain_event_received`) that are emitted
> for each detected blockchain event.
>
> For more information see the [Events](../events.html) reference section.

### URL

The base path for your Contract API is:

- `/api/v1/namespaces/{ns}/apis/{apiName}`

For the default namespace, this can be shortened to:

- `/api/v1/apis/{apiName}`

### FireFly Interface (FFI) and On-chain Location

Contract APIs are registered against:

1. A FireFly Interface (FFI) definition, which defines in a blockchain agnostic
   format the list of functions/events supported by the smart contract. Also 
   detailed type information about the inputs/outputs to those functions/events.

2. An optional `location` configured on the Contract API describes where the
   instance of the smart contract the API should interact with exists in the blockchain layer.
   For example the `address` of the Smart Contract for an Ethereum based blockchain,
   or the `name` and `channel` for a Hyperledger Fabric based blockchain.

If the `location` is not specified on creation of the Contract API, then it must be
specified on each API call made to the Contract API endpoints.

### OpenAPI V3 / Swagger Definitions

Each Contract API comes with an OpenAPI V3 / Swagger generated definition, which can
be downloaded from:

- `/api/v1/namespaces/{namespaces}/apis/{apiName}/api/swagger.json`

### Swagger UI

A browser / exerciser UI for your API is also available on:

- `/api/v1/namespaces/{namespaces}/apis/{apiName}/api`
