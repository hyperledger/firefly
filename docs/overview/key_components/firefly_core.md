---
layout: default
title: FireFly Core
parent: The Key Components
grand_parent: pages.understanding_firefly
nav_order: 2
---

# FireFly Core

{: .no_toc }

---

## FireFly Core Overview

FireFly Core acts as the brain for the Supernode. It is written in Go and hosts the API and UI. The two main components include the Event Bus and the orchestration layer. The Event Bus receives inputs from connected data sources, including on chain data, shared storage, and private data, as well as security and API layers.

The second portion is the Orchestration Engine which takes inputs from the Event Bus as well as user commands from the FireFly CLI (command line interface) and manages the lifecycle of assets and data. The Orchestration Engine then translates these inputs into data and state that is viewable in the FireFly Explorer.

![FireFly Core](../../images/firefly_core.png "FireFly Core")
