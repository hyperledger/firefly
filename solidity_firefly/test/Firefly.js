'use strict';

const FireflyContract = artifacts.require('./Firefly.sol');
const Payment = artifacts.require('./Payment.sol');
const {randomBytes} = require('crypto')

const initialSupply = 10000;

function randB32Hex() {
  return `0x${randomBytes(32).toString('hex')}`
}

contract('Firefly.sol', accounts => {
  let payment;
  let fireflyContract;

  before(async () => {
    payment = await Payment.new(initialSupply);
    fireflyContract = await FireflyContract.new(payment.address);
    await payment.approve(fireflyContract.address, initialSupply);
  });

  describe('Firefly', () => {

    describe('broadcastBatch', () => {

      it('broadcastBatch emits a BroadcastBatch event', async () => {
        const namespace = "ns1";
        const uuids = randB32Hex();
        const batchHash = randB32Hex();
        const payloadRef = randB32Hex();
        const contexts = [randB32Hex(),randB32Hex(),randB32Hex()];
        const result = await fireflyContract.pinBatch(namespace, uuids, batchHash, payloadRef, contexts);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.namespace, namespace);
        assert.equal(logArgs.uuids, uuids);
        assert.equal(logArgs.batchHash, batchHash);
        assert.equal(logArgs.payloadRef, payloadRef);
        assert.equal(logArgs.contexts.length, 3);
        assert.equal(logArgs.contexts[0], contexts[0]);
        assert.equal(logArgs.contexts[1], contexts[1]);
        assert.equal(logArgs.contexts[2], contexts[2]);
      });

    });


  });

});