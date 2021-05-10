# Kaleido Project Firefly

FireFly is a multiparty system for enterprise data flows, powered by blockchain. It solves all of the layers of complexity that sit between the low level blockchain and high level business processes and user interfaces. FireFly enables developers to build blockchain apps for enterprise up to 100x faster by allowing them to focus on business logic instead of infrastructure.

![Introducing Firefly](./architecture/intro_to_firefly_teaser.svg)

Please see the
[Hyperledger Firefly proposal document](https://docs.google.com/document/d/1o85YSowgCm226PEzdejbD2-3VQkrIwTdMCdpfXxsuQw/edit?usp=sharing)
for more information about the project goals an architecture.

## Navigating this repo

There are **two core codebases** in this repo, that have evolved through production 

### Kaleido Asset Trail (KAT): TypeScript - Generation 1

Directories:
- [kat](./kat): The core TypeScript runtime
- [solidity_kat](./solidity_kat): Ethereum/Solidity smart contract code
- [cordapp_kat](./cordapp_kat): The Corda smart contract (CorDapp)

This was the original implementation of the multi-party system API by Kaleido, and is already deployed in a number production projects.

The codebase distilled years of learning from enterprise blockchain projects Kaleido had been involved in, into a set of patterns for performing blockchain-orchestrated data exchange.

The persistence layer is a NoSQL based, and it is tightly integrated with the production grade services provided by Kaleido for streaming transactions onto the blockchain, and performing private data exchange of messages and documents.

As such it depends on the following Kaleido services:

- Blockchain nodes
  - Ethereum with the Kaleido [Kaleido REST API Gateway](https://docs.kaleido.io/kaleido-services/ethconnect/)
  - Corda with the Kaleido built-in API for streaming KAT transactions
- [Kaleido Event Streams](https://docs.kaleido.io/kaleido-services/event-streams/)
- [Kaleido App2App Messaging](https://docs.kaleido.io/kaleido-services/app2app/)
- [Kaleido Document Exchange](https://docs.kaleido.io/kaleido-services/document-store/)

### Firefly: Golang - Generation 2

Directories:
- [internal](./internal): The core implementation code
- [pkg](./pkg): Any libraries intended for external project use
- [cmd](./cmd): The command line entry point
- [solidity_firefly](./solidity_firefly): Ethereum/Solidity smart contract code

As the project evolved the acceleration it provides to enterprise projects became even clearer, and we wanted to widen the project to help provide that acceleration across all enterprise multi-party system development.

The value of widening to a fully fledged Open Source community became clear.

In doing this we made some fundamental engineering decisions:
- Move to Golang
  - High performance, fast starting, modern runtime
  - Great developer community
  - Many core runtime projects of similar type
- Move to a from active/passive to active/active HA architecture for the core runtime
  - Deferring to the core database for state high availability
  - Exploiting leader election where required
- Move to a fully pluggable architecture
  - Everything from Database through Blockchain, to Compute
  - Structured Golang plugin infrastructure to decouple the core code
  - Remote Agent model to decouple code languages, and HA designs
- Revisit the API resource model
  - A new set of nouns learning from developer experience:
 - `Asset`, `Data`, `Message`, `Event`, `Topic`, `Transaction`
- Add flexibility, while simplifying the developer experience:
  - Versioning for data schemas
  - Introducing a first class `Context` construct to help link together events
  - Allow many pieces of data to flow together, and be automatically re-assembled together
  - Clearer separation of concerns between the Firefly DB and the Application DB
  - Better search, filter and query support

## API Query Syntax

REST collections provide filter, `skip`, `limit` and `sort` support.
- The field in the message is used as the query parameter
- When multiple query parameters are supplied these are combined with AND
- When the same query parameter is supplied multiple times, these are combined with OR

### Example

`GET` `/api/v1/messages?confirmed=>0&type=broadcast&topic=t1&topic=t2&context=@someprefix&sort=sequence&descending&skip=100&limit=50`

This states:

- Filter on `confirmed` greater than 0
- Filter on `type` exactly equal to `broadcast`
- Filter on `topic` exactly equal to `t1` _or_ `t2`
- Filter on `context` containing the case-sensitive string `someprefix`
- Sort on `sequence` in `descending` order
- Paginate with `limit` of `50` and `skip` of `100` (e.g. get page 3, with 50/page)

Table of filter operations, which must be the first character of the query string (after the `=` in the above URL path example)

| Operator | Description                       |
|----------|-----------------------------------|
| (none)   | Equal                             |
| `!`      | Not equal                         |
| `<`      | Less than                         |
| `<=`     | Less than or equal                |
| `>`      | Greater than                      |
| `>=`     | Greater than or equal             |
| `@`      | Containing - case sensitive       |
| `!@`     | Not containing - case sensitive   |
| `^`      | Containing - case insensitive     |
| `!^`     | Not containing - case insensitive |

## Setup

Kaleido asset trail can be run as a Kaleido member service or as a standalone application.
For the latter, deploy an ERC20 token and use its address in the constructor of the [asset Trail smart contract](solidity_new/contracts/AssetTrail.sol),

For each participating member, deploy the following runtimes:
* IPFS
* App2App Messaging (with 2 destinations representing KAT and the client)
* Document Exchange (with 1 destination)

You must also define an Event Stream with subscriptions to all relevant
events for your use case (subscribe to all events if unsure).

Asset trail has built-in storage and can optionally be configured to use MongoDB.

Edit one of the configuration files in [core/data](core/data), or create a new folder for your config.
Populate `config.json` with the URLs for the deployed contract API, the event stream, the IPFS/App2App/Document
Exchange runtimes, a valid set of credentials, and the locally running MongoDB.

You can create separate config folders for each org you wish to simulate.

Run the server with the following (substitute the path to your own data directory as needed):
```
cd core
DATA_DIRECTORY=data/single-region/OrgA nodemon
```

If using Visual Studio Code, there is also a provided [.vscode/launch.json](launch.json) file which can be
edited to add launch configurations to the UI.
