"use strict";

import { ethers } from "hardhat";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { CustomPin } from "../typechain-types";
import { randomBytes } from "crypto";
import { assert } from "chai";
import {
  deployMockContract,
  MockContract,
} from "@ethereum-waffle/mock-contract";
import { abi as FireFlyABI } from "../../../../test/data/contracts/firefly/Firefly.json";

function randB32Hex() {
  return `0x${randomBytes(32).toString("hex")}`;
}

describe("CustomPin.sol", () => {
  let deployer: SignerWithAddress;
  let contract: CustomPin;
  let mockFireFly: MockContract;

  before(async () => {
    [deployer] = await ethers.getSigners();
    mockFireFly = await deployMockContract(deployer, FireFlyABI);
    const Factory = await ethers.getContractFactory("CustomPin");
    contract = await Factory.connect(deployer).deploy();
    await contract.deployed();
    await contract.setFireFlyAddress(mockFireFly.address);
  });

  it("sayHello", async () => {
    const data = randB32Hex();
    await mockFireFly.mock.pinBatchData.withArgs(data).returns();
    const result = await contract.sayHello(data);
    const receipt = await result.wait();
    assert.equal(receipt.events?.[0]?.event, "Hello");
  });
});
