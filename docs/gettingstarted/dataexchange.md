---
layout: default
title: Basic Data Exchange Application
parent: Getting Started
nav_order: 2
---

# Basic Data Exchange Application
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

To interact programmatically with some of the core FireFly APIs, we recommend trying out the fully-baked [FireFly Data Exchange](https://github.com/hyperledger-labs/firefly-dataexchange-https) application.  The available README contains straightforward instructions for generating the required configs and associated x509 certificates + metadata.  However, for an even smoother experience you can simply:

* Clone the repo and build the packages - `npm run build`
  - This will in turn generate two member identities, along with the required PKI material, for immediate usage - `Member-A` & `Member-B`
* Using an interactive IDE like Visual Studio, import the updated project and kick off the program for each member - `Member-A` & `Member-B`
* `Member-A`'s interactive Swagger is accessible locally on port 3000
* `Member-B`'s interactive Swagger is accessible locally on port 4000

If you see "Unauthorized" responses within the Swagger instance or through a REST client, be sure to update the API key value:

* In Swagger - click "Authorize" at the top of the screen and input `xxxxx` as the api key.
* Via a REST client - create an `x-api-key` header with `xxxxx` as the value
