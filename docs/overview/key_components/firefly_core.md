---
layout: default
title: FireFly Core
parent: The Key Components
grand_parent: pages.understanding_firefly
nav_order: 2
---

# FireFly Core
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## FireFly Core

FireFly Core acts as the brain for the SuperNode. There are two main components which is the Orchestration Engine, Event Bus, Data, and State management.

Support for multiple protocols across public and private chains with a powerful orchestration engine backed by a scalable event bus. Manage private data, on-chain data, and shared storage. Track state across networks, chains, and internal back office systems.

### Data

### Orchestration Engine

## Pluggable microservices architecture

The runtimes are pluggable, allowing technology choice, and extensibility.

- FireFly Core
  - Orchestration engine - manages lifecycle of assets and data
  - Hosts the API and UI - applications connect here
  - Maintains private storage
  - Written in Go
- Connectors
  - Runtimes that bridge the core to multi-party infrastructure
  - Can be written in any language Go, Java, Node.js etc.
  - Can be stateful or stateless, depending on requirements
  - Can contain significant function, such as managed file transfer, or e2e encryption
- Infrastructure runtimes
  - Can be local runtimes, or cloud services
  - Blockchain nodes - Fabric, Ethereum, Corda etc.
  - Database servers - PostgreSQL, SQLite, CouchDB etc.
  - Private messaging - Kafka, RabbitMQ, ActiveMQ, Mosquitto etc.
  - Private blob storage - Kubernetes PVCs, AWS S3, Azure File etc.
  - Public blob storage - IPFS, etc.
  - ... and more - token bridges, trusted compute engines, etc.