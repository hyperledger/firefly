---
layout: i18n_page
title: pages.faqs
nav_order: 9
has_children: false
---

# FAQs

Find answers to the most commonly asked FireFly questions.

## How does FireFly enable multi-chain applications?
It's best to think about FireFly as a rich orchestration layer that sits one layer above the blockchain.  FireFly helps to abstract away much of the complex blockchain functionality (such as data exchange, private messaging, common token functionality, etc) in a loosely coupled microservice architecture with highly pluggable components. This enables application developers to focus on building innovative Web3 applications.

There aren't any out of the box bridges to connect two separate chains together, but with a collection of FireFly instances across a consortium, FireFly could help listen for events on Blockchain A and take an action on Blockchain B when certain conditions are met.

## 📜 How do I deploy smart contracts?
In order to interact with a smart contract on a public or private Ethereum chain you need to first deploy it to the chain. Deployment means that you've sent a transaction with the compiled source code to the chain without a specified recipient and received a contract address that you and others on the network can use to interact with your contract.

Refer to the [Smart Contract Tutorial](../tutorials/custom_contracts/) for instructions to deploy a smart contract to your private networking using FireFly.

## 🦊 Can I connect FireFly to MetaMask?
Yes! Before you set up MetaMask you'll likely want to create some tokens that you can use to send between wallets on your FF network. Go to the tokens tab in your FireFly node's UI, create a token pool, and then mint some tokens. Once you've done this, follow the steps listed [here](https://hyperledger.github.io/firefly/tutorials/tokens/erc721.html#use-metamask) to set up MetaMask on your network.

## 🚀 Connect with us on Discord
If your question isn't answered here or if you have immediate questions please don't hesitate to reach out to us on [Discord](https://discord.gg/hyperledger_) in the `firefly` channel: