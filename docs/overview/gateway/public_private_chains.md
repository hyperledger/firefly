---
layout: default
title: Public vs. Private chains
parent: pages.web3_gateway_features
grand_parent: pages.understanding_firefly
nav_order: 1
---

# Public vs. Private Chains
{: .no_toc }

---

## Public and Permissioned Blockchain

A separate choice to the technology for your blockchain, is what combination
of blockchain ecosystems you will integrate with.

There are a huge variety of options, and increasingly you might find yourself
integrating with multiple ecosystems in your solutions.

A rough (and incomplete) high level classification of the blockchains available is as follows:

- Layer 1 public blockchains
  - This is where most token ecosystems are rooted and some of the popular ones include Ethereum, Avalanche, BNB Chain, etc.)
- Layer 2 public scaling solutions backed by a Layer 1 blockchain
  - These are increasing where transaction execution takes place that
    needs to be reflected eventually back to a Layer 1 blockchain (due
    to cost/congestion in the Layer 1 chains). An example of a Layer 2 solution would be Arbitrum which uses rollups to connect back to Ethereum.
- Permissioned side-chains
  - Historically this has been where the majority of production adoption of
    enterprise blockchain has focussed, due to the predictable cost, performance,
    and ability to manage the validator set and boundary API security
    alongside a business network governance policy
  - These might have their state check-pointed/rolled-up to a Layer 2 or Layer 1 chain

The lines are blurring between these categorizations as the technologies and ecosystems evolve.

## Public blockchain variations

For the public Layer 1 and 2 solutions, there are too many subclassifications to go into in detail here:

- Whether ecosystems supports custom smart contract execution (EVM based is most common, where contracts are supported)
- What types of token standards are supported, or other chain specific embedded smart contracts
- Whether the chain follows an unspent transaction output (UTXO) or Account model
- How value is bridged in-to / out-of the ecosystem
- How the validator set of the chain is established - most common is Proof of Stake (PoS)
- How data availability is maintained - to check the working of the validators ensure the historical state is not lost
- The consensus algorithm, and how it interacts with the consensus of other blockchains
- How state in a Layer 2 is provable in a parent Layer 1 chain (rollup technologies etc.)

## Common public considerations

The thing most consistent across public blockchain technologies, is that the technical decisions are
backed by token economics.

Put simply, creating a system where it's more financially rewarding to behave honestly, than it
is to subvert and cheat the system.

This means that participation costs, and that the mechanisms needed to reliably get your transactions
into these systems are complex. Also that the time it might take to get a transaction onto the chain
can be much longer than for a permissioned blockchain, with the potential to have to make a number
of adjustments/resubmissions.

The choice of whether to run your own node, or use a managed API, to access these blockchain ecosystems
is also a factor in the behavior of the transaction submission and event streaming.
