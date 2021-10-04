---
layout: default
title: Install the FireFly CLI
parent: Getting Started
nav_order: 1
---

# Install the FireFly CLI
{: .no_toc }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## FireFly CLI

The FireFly CLI helps you start and administer a local FireFly development environment.

## References and Dependencies

* [CLI Repo README](https://github.com/hyperledger/firefly-cli)
* [Go](https://golang.org/doc/install) - latest version recommended (v1.16.5)
  - If you haven't installed Go previously, make sure to add to your `$PATH` env variable - `$HOME/go/bin`
* [Docker](https://docs.docker.com/docker-for-mac/install/) - the CLI will download the requisite FireFly images and bootstrap a collection of containers (Ethereum adapter, FireFly Core, IPFS, local Ganache blockchain, postgres, etc.)

## Get the CLI

> Binary distributions coming soon - see https://github.com/hyperledger/firefly/issues/77

* On Go 1.16 or newer
  - `go install github.com/hyperledger/firefly-cli/ff@latest`
* On earlier versions of Go
  - `go get github.com/hyperledger/firefly-cli/ff`

