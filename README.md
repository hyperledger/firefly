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
