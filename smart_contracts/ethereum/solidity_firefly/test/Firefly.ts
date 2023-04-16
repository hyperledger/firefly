"use strict";

import { ethers } from "hardhat";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { Firefly } from "../typechain-types";
import { randomBytes } from "crypto";
import { assert } from "chai";

function randB32Hex() {
  return `0x${randomBytes(32).toString("hex")}`;
}

describe("Firefly.sol", () => {
  let deployer: SignerWithAddress;
  let fireflyContract: Firefly;

  before(async () => {
    [deployer] = await ethers.getSigners();
    const Factory = await ethers.getContractFactory("Firefly");
    fireflyContract = await Factory.connect(deployer).deploy();
    await fireflyContract.deployed();
  });

  describe("Firefly", () => {
    describe("pinBatch", () => {
      it("broadcast with a payloadRef", async () => {
        const uuids = randB32Hex();
        const batchHash = randB32Hex();
        const payloadRef = "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD";
        const contexts = [randB32Hex(), randB32Hex(), randB32Hex()];
        const result = await fireflyContract.pinBatch(
          uuids,
          batchHash,
          payloadRef,
          contexts
        );
        const receipt = await result.wait();
        const logArgs = receipt.events?.[0]?.args;
        assert.isDefined(logArgs);
        if (logArgs) {
          assert.equal(logArgs.author, deployer.address);
          assert.equal(logArgs.action, "firefly:batch_pin");
          assert.equal(logArgs.uuids, uuids);
          assert.equal(logArgs.batchHash, batchHash);
          assert.equal(logArgs.payloadRef, payloadRef);
          assert.equal(logArgs.contexts.length, 3);
          assert.equal(logArgs.contexts[0], contexts[0]);
          assert.equal(logArgs.contexts[1], contexts[1]);
          assert.equal(logArgs.contexts[2], contexts[2]);
        }
      });

      it("private with an empty payloadRef", async () => {
        const uuids = randB32Hex();
        const batchHash = randB32Hex();
        const payloadRef = "";
        const contexts = [randB32Hex(), randB32Hex(), randB32Hex()];
        const result = await fireflyContract.pinBatch(
          uuids,
          batchHash,
          payloadRef,
          contexts
        );
        const receipt = await result.wait();
        const logArgs = receipt.events?.[0]?.args;
        assert.isDefined(logArgs);
        if (logArgs) {
          assert.equal(logArgs.author, deployer.address);
          assert.equal(logArgs.action, "firefly:batch_pin");
          assert.equal(logArgs.uuids, uuids);
          assert.equal(logArgs.batchHash, batchHash);
          assert.equal(logArgs.payloadRef, payloadRef);
          assert.equal(logArgs.contexts.length, 3);
          assert.equal(logArgs.contexts[0], contexts[0]);
          assert.equal(logArgs.contexts[1], contexts[1]);
          assert.equal(logArgs.contexts[2], contexts[2]);
        }
      });
    });

    describe("pinBatchData", () => {
      it("broadcast with a payloadRef", async () => {
        const uuids = randB32Hex();
        const batchHash = randB32Hex();
        const payloadRef = "Qmf412jQZiuVUtdgnB36FXFX7xg5V6KEbSJ4dpQuhkLyfD";
        const contexts = [randB32Hex(), randB32Hex(), randB32Hex()];
        const abiCoder = ethers.utils.defaultAbiCoder;
        const data = abiCoder.encode(
          ["bytes32", "bytes32", "string", "bytes32[]"],
          [uuids, batchHash, payloadRef, contexts]
        );
        const result = await fireflyContract.pinBatchData(data);
        const receipt = await result.wait();
        const logArgs = receipt.events?.[0]?.args;
        assert.isDefined(logArgs);
        if (logArgs) {
          assert.equal(logArgs.author, deployer.address);
          assert.equal(logArgs.action, "firefly:contract_invoke_pin");
          assert.equal(logArgs.uuids, uuids);
          assert.equal(logArgs.batchHash, batchHash);
          assert.equal(logArgs.payloadRef, payloadRef);
          assert.equal(logArgs.contexts.length, 3);
          assert.equal(logArgs.contexts[0], contexts[0]);
          assert.equal(logArgs.contexts[1], contexts[1]);
          assert.equal(logArgs.contexts[2], contexts[2]);
        }
      });
    });

    describe("networkAction", () => {
      it("terminate action", async () => {
        const result = await fireflyContract.networkAction(
          "firefly:terminate",
          "123"
        );
        const receipt = await result.wait();
        const logArgs = receipt.events?.[0]?.args;
        assert.isDefined(logArgs);
        if (logArgs) {
          assert.equal(logArgs.author, deployer.address);
          assert.equal(logArgs.action, "firefly:terminate");
          assert.equal(
            logArgs.uuids,
            "0x0000000000000000000000000000000000000000000000000000000000000000"
          );
          assert.equal(
            logArgs.batchHash,
            "0x0000000000000000000000000000000000000000000000000000000000000000"
          );
          assert.equal(logArgs.payloadRef, "123");
          assert.equal(logArgs.contexts.length, 0);
        }
      });
    });
  });
});
