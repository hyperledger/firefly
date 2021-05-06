# Kaleido Project Firefly

# E2E Event Flow

## REFACTOR IN PROGRESS

In the evolution from Kaleido Asset Trail to Project Firefly, we are taking the opportunity to make significant changes:

- Code implementation moving from Typescript to Go
- Renaming nouns on the API surface are based on learning from production projects
- Enhancing the pluggability of the project

## Component Architecture Overview

![Project Firefly Component Architecture](architecture/firefly_component_architecture.jpg)

## Component E2E flow example

![Project Firefly E2E flow example](architecture/firefly_flow_overview_sequencediagram.png)

## Kaleido Asset Trail (KAT) background

![Project Firefly](firefly_overview.png)

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
