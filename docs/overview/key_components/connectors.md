---
layout: i18n_page
title: pages.connector_framework
parent: pages.key_features
grand_parent: pages.understanding_firefly
nav_order: 3
---

# Connector Framework

{: .no_toc }

---

![Hyperledger FireFly Connectivity Features](../../images/firefly_functionality_overview_connectivity.png)

## Pluggable Microservices Architecture

The ability for every component to be pluggable is at the core of Hyperledger FireFly.

A microservices approach is used, combining code plug-points in the core runtime, with API extensibility
to remote runtimes implemented in a variety of programming languages.

[![Hyperledger FireFly Architecture Overview](../../images/firefly_architecture_overview.jpg)](../../images/firefly_architecture_overview.jpg)

## Extension points

- Blockchain - a rich framework for extensibility to any blockchain / digital ledger technology (DLT)
- Tokens - mapping token standards and governance models to a common data model
- Shared storage - supporting permissioned and public distributed storage technologies
- Data exchange - private local/storage and encrypted transfer of data
- Identity - flexibility for resolving identities via Decentralized IDentifier (DID)
- Persistence - the local private database

> Learn more about the [plugin architecture here](../../architecture/plugin_architecture.html)

## Blockchain Connector Framework

The most advanced extension point is for the blockchain layer, where multiple layers of extensibility
are provided to support the programming models, and behaviors of different blockchain technologies.

This framework has been proven with technologies as different as EVM based Layer 2 Ethereum Scaling
solutions like Polygon, all the way to permissioned Hyperledger Fabric networks.

> Check out instructions to connect to a list of remote blockchain networks [here](../../tutorials/chains/).

![FireFly Blockchain Connector Framework](../../images/firefly_blockchain_connector_framework.png)

Find out more about the Blockchain Connector Framework [here](../../architecture/blockchain_connector_framework.html).