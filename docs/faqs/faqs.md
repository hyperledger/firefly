---
layout: i18n_page
title: pages.faqs
nav_order: 9
has_children: true
---

# FAQs

Find answers to the most commonly asked FireFly questions.

## How does FireFly enable multi-chain applications?
It's best to think about FireFly as a rich orchestration layer that sits one layer above the blockchain.  FireFly helps to abstract away much of the complex blockchain functionality (such as data exchange, private messaging, common token functionality, etc) in a loosely coupled microservice architecture with highly pluggable components. This enables application developers to focus on building innovative Web3 applications.

There aren't any out of the box bridges to connect two separate chains together, but with a collection of FireFly instances across a consortium, FireFly could help listen for events on Blockchain A and take an action on Blockchain B when certain conditions are met.

## 📜 How do I deploy smart contracts?
In order to interact with a smart contract on a public or private Ethereum chain you need to first deploy it to the chain. Deployment means that you've sent a transaction with the compiled source code to the chain without a specified recipient and received a contract address that you and others on the network can use to interact with your contract.

In order to deploy the contract on the Ethereum network, you'll need the following:
- Your contract's bytecode (which is generated through [compilation](https://ethereum.org/en/developers/docs/smart-contracts/compiling/_))
- Any currency needed to cover gas fees on your chosen network (i.e. ETH for gas on the public Ethereum network)
- A deployment script (two of the most popular deployment tools are [Hardhat](https://hardhat.org/guides/deploying.html) and [Truffle](https://trufflesuite.com/docs/truffle/advanced/networks-and-app-deployment/))
- Access to a node on your network

Once your contract is deployed, you can view the contract details through popular blockchain explorers such as [Etherscan](https://etherscan.io/), which will show all transaction interactions with the contract and show any token value associated with the contract.

For additional information about Smart Contracts, please see the official [Ethereum.org documentation](https://ethereum.org/en/developers/docs/smart-contracts/)

## 🦊 Can I connect FireFly to MetaMask?
Yes! In order to do this, you'll want to set up a FireFly stack and deploy an ERC-20 or ERC-721 contract to the chain (see the FAQ above on how to deploy a smart contract). 

Once this contract is deployed, follow the steps listed [here](https://hyperledger.github.io/firefly/tutorials/tokens/erc721.html#use-metamask)

## 🚀 Connect with us on Discord
If your question isn't answered here or if you have immediate questions please don't hesitate to reach out to us on [Discord](https://discord.gg/hyperledger_) in the `firefly` channel: