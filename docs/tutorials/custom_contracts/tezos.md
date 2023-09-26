---
layout: default
title: Tezos
parent: pages.custom_smart_contracts
grand_parent: pages.tutorials
nav_order: 3
---

# Work with Tezos smart contracts

{: .no_toc }
This guide describes the steps to deploy a smart contract to a Tezos blockchain and use FireFly to interact with it in order to submit transactions, query for states and listening for events.

## Table of contents

{: .no_toc .text-delta }

1. TOC
   {:toc}

---

## Smart Contract Languages

Smart contracts on Tezos can be programmed using familiar, developer-friendly languages. All features available on Tezos can be written in any of the high-level languages used to write smart contracts, such as Archetype, LIGO, and SmartPy. These languages all compile down to Michelson and you can switch between languages based on your preferences and projects.

> **NOTE:** For this tutorial we are going to use [SmartPy](https://smartpy.io/) for building Tezos smart contracts utilizing the broadly adopted Python language.

## Example smart contract

First let's look at a simple contract smart contract called `SimpleStorage`, which we will be using on a Tezos blockchain. Here we have one state variable called 'storedValue' and initialized with the value 12. During initialization the type of the variable was defined as 'int'. You can see more at [SmartPy types](https://smartpy.io/manual/syntax/integers-and-mutez). And then we added a simple test, which set the storage value to 15 and checks that the value was changed as expected.

> **NOTE:** Tests are used to verify the validity of contract entrypoints and do not affect the state of the contract during deployment.

Here is the source for this contract:

```smartpy
import smartpy as sp

@sp.module
def main():
    class SimpleStorage(sp.Contract):
        def __init__(self, value):
            self.data.storedValue = value

        @sp.entrypoint
        def replace(self, params):
            self.data.storedValue = params.value

@sp.add_test(name="SimpleStorage")
def test():
    c1 = main.SimpleStorage(12)
    scenario = sp.test_scenario(main)
    scenario.h1("SimpleStorage")
    scenario += c1
    c1.replace(value=15)
    scenario.verify(c1.data.storedValue == 15)
```

