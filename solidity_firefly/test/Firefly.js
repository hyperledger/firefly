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
        const txnId = randB32Hex();
        const batchId = randB32Hex();
        const payloadRef = randB32Hex();
        const result = await fireflyContract.broadcastBatch(txnId, batchId, payloadRef);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.txnId, txnId);
        assert.equal(logArgs.batchId, batchId);
        assert.equal(logArgs.payloadRef, payloadRef);
      });

    });


  });

});