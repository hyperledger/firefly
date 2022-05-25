---
layout: default
title: FAQs
nav_order: 9
has_children: false
---

# FAQ
{: .no_toc }

---

Find answers to the most commonly asked FireFly questions.

## How does FireFly enable multi-chain applications?
FireFly is designed as a loosely coupled microservice architecture with highly pluggable components. Together, these services abstract away much of the complex blockchain functionality (such as data exchange, private messaging, common token functionality, etc) so that application developers can focus on building innovative Web3 applications. There isn't an out of the box bridge, but 

<!-- We can reconnect for sure.  Think of FireFly as a rich orchestration layer that actually sits above your blockchain.  It helps abstract really important things like private data exchange, messaging, broadcasts, token interfaces, smart contract interfaces, events, etc...  It's a loosely coupled microservice architecture with highly pluggable components (bring your own database or PKI material).

So you can liken FireFly to something analogous to an organizational Gateway that integrates with your critical systems and existing stack.  So you could run a collection of FF instances across a consortium or company and pretty easily facilitate cross-chain functionality or basic interop.  There isn't a bridge out of the box, but it's really easy to listen to Blockchain A and then react on Blockchain B, for example. -->

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
