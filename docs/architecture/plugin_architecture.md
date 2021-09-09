---
layout: default
title: Plugin Architecture
parent: Architecture
nav_order: 4
---

# Plugin Architecture
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

<iframe width="736" height="414" src="https://www.youtube.com/embed/wkuQjBy_uhg" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

---
![FireFly Plugin Architecture](../images/firefly_plugin_architecture.svg "FireFly Plugin Architecture")
This diagram shows the various plugins that are currently in the codebase and the layers in each plugin

---

![FireFly Plugin Architecture](../images/firefly_plugin_architecture.jpg "FireFly Plugin Architecture")
This diagram shows the details of what goes into each layer of a FireFly plugin

---

## Overview
The FireFly node is built for extensibility, with separate pluggable runtimes orchestrated into a common API for developers.  The mechanics of that 
pluggability for developers of new connectors is explained below:

This architecture is designed to provide separations of concerns to account for:
- Differences in code language for the low-level connection to a backend (Java for Corda for example)
- Differences in transports, particularly for delivery of events:
  - Between FireFly Core and the Connector
    - Different transports other than HTTPS/WebSockets (GRPC etc.), and different wire protocols (socket.io, etc.)
  - Between the Connector and the underlying Infrastructure Runtime
     - Often this is heavy lifting engineering within the connector
- Differences in High Availability (HA) / Scale architectures
   - Between FireFly Core, and the Connector
     - Often for event management, and active/passive connector runtime is sufficient
   - Between the Connector and the Infrastructure Runtime
     - The infrastructure runtimes have all kinds of variation here... think of the potential landscape here from PostreSQL through Besu/Fabric/Corda, to Hyperledger Avalon and even Main-net ethereum

## FireFly Core

- Golang
- N-way scalable cluster
  - Database is also pluggable via this architecture
- No long lived in-memory processing
  - All micro-batching must be recoverable
- Driven by single configuration set
  - Viper semantics - file, env var, cmdline flags

## Plugin for Connector

- Golang
- Statically compiled in support at runtime
  - Go dynamic plugin support too immature
- Must be 100% FLOSS code (no GPL/LGPL etc.)
- Contributed via PR to FF Core
- Intended to be lightweight binding/mapping
- Must adhere to FF Core Coding Standards
- Scrutiny on addition of new frameworks/transports

## Connector

- Node.js / Java / Golang, etc.
- Runs/scales independently from FF core
- Coded in any language, OSS or proprietary
- One runtime or multiple
- HA model can be active/passive or active/active
- Expectation is all plugins need a connector
  - Some exceptions exist (e.g. database plugin)

## Infrastructure Runtime

- Besu, Quorum, Corda, Fabric, IPFS, Kafka, etc.
- Runs/scales independently from FF Core 
- Coded in any language, OSS or proprietary 
- Not specific to FireFly
- HA model can be active/passive or active/active
