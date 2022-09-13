---
layout: i18n_page
title: pages.web3_gateway_features
parent: pages.understanding_firefly
nav_order: 4
has_children: true
---

# Gateway Mode

{: .no_toc }

## Table of contents

{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Introduction

pages.web3_gateway_features allows your FireFly Supernode to connect to any blockchain ecosystem, public or private. When a chain is connected, the FireFly Supernode may invoke custom smart contracts, interact with tokens, and monitor transactions. A single FireFly Supernode is able to have multiple namespaces, or isolated environments, where each namespace is a connection to a different chain.

![Gateway Mode](../../images/gateway_mode.png "Gateway Mode")

## Use Case Example

There are many ways to use Gateway Mode. An example could be an interaction with a public chain such as Polygon.

Imagine as a developer that there are multiple smart contracts that you have written for use on Polygon. You can start with FireFly's API generation to have a easy to use REST API interface to interact with the smart contracts. Next, imagine that one or more of your smart contracts conforms to the ERC-20 standard. Your Supernode is able to index the token operations (mint/burn/transfer/approvals) to see transaction history and balances associated with that contract. Finally, using FireFly’s tokens api, you may further interact with that ERC20.