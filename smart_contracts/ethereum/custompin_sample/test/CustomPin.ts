"use strict";

import { ethers } from "hardhat";
import { SignerWithAddress } from "@nomicfoundation/hardhat-ethers/signers";
import { CustomPin } from "../typechain-types";
import { randomBytes } from "crypto";
import { assert } from "chai";
import { ContractFactory, AbiCoder } from "ethers";
import {
  abi as FireflyABI,
  bytecode as FireflyBytecode,
} from "../../../../test/data/contracts/firefly/Firefly.json";

function randB32Hex() {
  return `0x${randomBytes(32).toString("hex")}`;
}

describe("CustomPin.sol", () => {
  let deployer: SignerWithAddress;
  let contract: CustomPin;

  before(async () => {
    [deployer] = await ethers.getSigners();
    const FireflyFactory = new ContractFactory(FireflyABI, FireflyBytecode);
    const fireflyContract = await FireflyFactory.connect(deployer).deploy();
    await fireflyContract.waitForDeployment();
    console.log("Firefly address: " + (await fireflyContract.getAddress()));

    const Factory = await ethers.getContractFactory("CustomPin");
    contract = await Factory.connect(deployer).deploy();
    await contract.waitForDeployment();
    await contract.setFireFlyAddress(await fireflyContract.getAddress());
  });

  it("sayHello", async () => {
    const uuids = randB32Hex();
    const batchHash = randB32Hex();
    const payloadRef = "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD";
    const contexts = [randB32Hex(), randB32Hex()];
    const abiCoder = AbiCoder.defaultAbiCoder();
    const data = abiCoder.encode(
      ["bytes32", "bytes32", "string", "bytes32[]"],
      [uuids, batchHash, payloadRef, contexts]
    );
    const result = await contract.sayHello(data);
    const receipt = await result.wait();
    assert.equal(receipt.logs[0].fragment.name, "Hello");
  });
});
