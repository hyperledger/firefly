'use strict';

const AssetTrailContract = artifacts.require('./AssetTrail.sol');
const Payment = artifacts.require('./Payment.sol');

const initialSupply = 10000;

/// Member constants

const testMemberNames = [
  'member-1',
  'member-2',
  'member-2-update'
];

const testAssetTrailInstanceIDs = [
  'xxxxxxxxxx-xxxxxxxxxx'
];

const testApp2AppDestinations = [
  'kld://app2app-destination-1',
  'kld://app2app-destination-2',
  'kld://app2app-destination-2-update'
];

const testDocExchangeDestinations = [
  'kld://documentstore-destination-1',
  'kld://documentstore-destination-2',
  'kld://documentstore-destination-2-update'
];

// Asset definition constants

const testAssetDefinitionIDs = [
  '0x6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b',
  '0xd4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35',
  '0x4e07408562bedb8b60ce05c1decfe3ad16b72230967de01f640b7e4729b49fce',
  '0x4b227777d4dd1fc61c6f884f48641d02b4d121d3fd328cb08b5531fcacdabf8a'
];

const testAssetDefinitionHashes = [
  '0x6ab9f1eb8f7d3388f4f9d586f66e99fd54080df2c446f0e58668b09c08a16dd0'
];

// Asset instance constants

const testAssetInstanceBatchHashes = [
  '0xa15abc51152dad05919a79966dc784f37491fa1c394b7935afcc5516b65ec98c',
  '0xd5cc610cc1373b3f26238641e51bf4d3cb20068ea3f589bfac413417b4d46f23'
];

const testAssetInstanceIDs = [
  '0x1ebb3de9307e8c992e7b18f1f48767f6b83eb6befb72985a4f3043609ffa1e96',
  '0x82459a7f3e4b132093db4d651c74e384e72f8a808c2b321d7b3343d6fb872187'
];

const testDescriptionHashes = [
  '0x27a84712e4b22c415fc544d55cdee82327a829f96d03329457f76ebf9af4dcaa',
  '0x13609d74cc8ea07555856a54ba51b01f831af4af89bd39847babe5bf6cb665df'
];

const testContentHashes = [
  '0x4fb431659a5b45f4e7b1a69bacb4101a11b82777de6857c3e40d7fe217307285',
  '0x1e335362351e60f908f58ac674f8c0967dca8af41c3b696066b49810d399d795'
];

// Payment definition constants

const testPaymentDefinitionIDs = [
  '0xf64551fcd6f07823cb87971cfb91446425da18286b3ab1ef935e0cbd7a69f68a',
  '0x3946ca64ff78d93ca61090a437cbb6b3d2ca0d488f5f9ccf3059608368b27693'
];

const testPaymentDefinitionNames = [
  'My subscriptions',
  'My purchases'
];

const testPaymentSchemas = [
  '0x6dee8ee9d5a7743e2a86e03e652f072f520d0955d6c7551ba4c85f71011d0896'
];

// Payment instance constants

const testPaymentInstanceIDs = [
  '0x38f31ec46e5bcf5f86502bb4985963c7cf5b821f60ca140c2562636a136dd376',
  '0xc18a1517995c76e2104fb6d977661a2b2322340cd92d2c98af4779cf8a92bb3a'
];

const testPaymentHashes = [
  '0x6c2d640aaf679ecf258150e2ceb742c48e8c1e71391d05a75e5e041ffd51c377'
]

// Asset property constants

const testAssetPropertyKeys = [
  'test-key'
];

const testAssetPropertyValues = [
  'test-value'
];

contract('AssetTrail.sol', accounts => {
  let payment;
  let assetTrailContract;

  before(async () => {
    payment = await Payment.new(initialSupply);
    assetTrailContract = await AssetTrailContract.new(payment.address);
    await payment.approve(assetTrailContract.address, initialSupply);
  });

  describe('Asset Trail', () => {

    describe('Members', () => {

      it('registerMember should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.registerMember('', testAssetTrailInstanceIDs[0], testApp2AppDestinations[0], testDocExchangeDestinations[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('registerMember should register a member and emit the corresponding event (member 1)', async () => {
        const result = await assetTrailContract.registerMember(testMemberNames[0], testAssetTrailInstanceIDs[0], testApp2AppDestinations[0], testDocExchangeDestinations[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.member, accounts[0]);
        assert.equal(logArgs.name, testMemberNames[0]);
        assert.equal(logArgs.assetTrailInstanceID, testAssetTrailInstanceIDs[0]);
        assert.equal(logArgs.app2appDestination, testApp2AppDestinations[0]);
        assert.equal(logArgs.docExchangeDestination, testDocExchangeDestinations[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('registerMember should register a member and emit the corresponding event (member 2)', async () => {
        const result = await assetTrailContract.registerMember(testMemberNames[1], testAssetTrailInstanceIDs[0], testApp2AppDestinations[1], testDocExchangeDestinations[1], { from: accounts[1] });
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.member, accounts[1]);
        assert.equal(logArgs.name, testMemberNames[1]);
        assert.equal(logArgs.assetTrailInstanceID, testAssetTrailInstanceIDs[0]);
        assert.equal(logArgs.app2appDestination, testApp2AppDestinations[1]);
        assert.equal(logArgs.docExchangeDestination, testDocExchangeDestinations[1]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('registerMember should allow members to update their name and destinations (member 2)', async () => {
        const result = await assetTrailContract.registerMember(testMemberNames[2], testAssetTrailInstanceIDs[0], testApp2AppDestinations[2], testDocExchangeDestinations[2], { from: accounts[1] });
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.member, accounts[1]);
        assert.equal(logArgs.name, testMemberNames[2]);
        assert.equal(logArgs.assetTrailInstanceID, testAssetTrailInstanceIDs[0]);
        assert.equal(logArgs.app2appDestination, testApp2AppDestinations[2]);
        assert.equal(logArgs.docExchangeDestination, testDocExchangeDestinations[2]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Assets definitions', () => {

      it('createAssetDefiniton should create a new asset definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createAssetDefinition(testAssetDefinitionHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionHash, testAssetDefinitionHashes[0]);
      });

    });

    describe('Asset instances', () => {

      it('createDescribedAssetInstance should create a new described asset instance and emit the corresponding event', async () => {
        const result = await assetTrailContract.createDescribedAssetInstance(testAssetInstanceIDs[0], testAssetDefinitionIDs[0], testDescriptionHashes[0], testContentHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetInstanceID, testAssetInstanceIDs[0]);
        assert.equal(logArgs.assetDefinitionID, testAssetDefinitionIDs[0]);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.descriptionHash, testDescriptionHashes[0]);
        assert.equal(logArgs.contentHash, testContentHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('createAssetInstance should create a new asset instance and emit the corresponding event', async () => {
        const result = await assetTrailContract.createAssetInstance(testAssetInstanceIDs[1], testAssetDefinitionIDs[2], testContentHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetInstanceID, testAssetInstanceIDs[1]);
        assert.equal(logArgs.assetDefinitionID, testAssetDefinitionIDs[2]);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.contentHash, testContentHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Asset instance batches', () => {

      it('createAssetInstanceBatch should emit the corresponding event', async () => {
        const result = await assetTrailContract.createAssetInstanceBatch(testAssetInstanceBatchHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.batchHash, testAssetInstanceBatchHashes[0]);
        assert.equal(logArgs.author, accounts[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Described payment definitions', () => {

      it('createDescribedPaymentDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedPaymentDefinition(testPaymentDefinitionIDs[0], '', testPaymentSchemas[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createDescribedPaymentDefinition should create a new payment definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createDescribedPaymentDefinition(testPaymentDefinitionIDs[0], testPaymentDefinitionNames[0], testPaymentSchemas[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.paymentDefinitionID, testPaymentDefinitionIDs[0]);
        assert.equal(logArgs.name, testPaymentDefinitionNames[0]);
        assert.equal(logArgs.descriptionSchemaHash, testPaymentSchemas[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });


    describe('Payment definitions', () => {

      it('createPaymentDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentDefinition(testPaymentDefinitionIDs[1], '');
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createPaymentDefinition should create a new payment definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createPaymentDefinition(testPaymentDefinitionIDs[1], testPaymentDefinitionNames[1]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.paymentDefinitionID, testPaymentDefinitionIDs[1]);
        assert.equal(logArgs.name, testPaymentDefinitionNames[1]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Described payment instances', () => {

      it('createDescribedPaymentInstance should raise an error if author and member are the same', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedPaymentInstance(testPaymentInstanceIDs[0], testPaymentDefinitionIDs[0], accounts[0], 1, testPaymentHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Author and member cannot be the same'));
      });

      it('createDescribedPaymentInstance should raise an error if amount is 0', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedPaymentInstance(testPaymentInstanceIDs[0], testPaymentDefinitionIDs[0], accounts[1], 0, testPaymentHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Amount must be greater than 0'));
      });

      it('createDescribedPaymentInstance should create a new payment instance and emit the corresponding event', async () => {
        const balanceAccount0Before = await payment.balanceOf(accounts[0]);
        const balanceAccount1Before = await payment.balanceOf(accounts[1]);

        const result = await assetTrailContract.createDescribedPaymentInstance(testPaymentInstanceIDs[0], testPaymentDefinitionIDs[0], accounts[1], 1, testPaymentHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.paymentInstanceID, testPaymentInstanceIDs[0]);
        assert.equal(logArgs.paymentDefinitionID, testPaymentDefinitionIDs[0]);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.member, accounts[1]);
        assert.equal(logArgs.amount, 1);
        assert.equal(logArgs.descriptionHash, testPaymentHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);

        const balanceAccount0After = await payment.balanceOf(accounts[0]);
        const balanceAccount1After = await payment.balanceOf(accounts[1]);
        assert(balanceAccount0Before.toNumber() === balanceAccount0After.toNumber() + 1);
        assert(balanceAccount1Before.toNumber() === balanceAccount1After.toNumber() - 1);
      });

    });

    describe('Payment instances', () => {

      it('createPaymentInstance should raise an error if author and member are the same', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentInstance(testPaymentInstanceIDs[0], testPaymentDefinitionIDs[1], accounts[0], 1);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Author and member cannot be the same'));
      });

      it('createPaymentInstance should raise an error if the amount is 0', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentInstance(testPaymentInstanceIDs[0], testPaymentDefinitionIDs[1], accounts[1], 0);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Amount must be greater than 0'));
      });

      it('createPaymentInstance should create a new payment instance and emit the corresponding event', async () => {
        const balanceAccount0Before = await payment.balanceOf(accounts[0]);
        const balanceAccount1Before = await payment.balanceOf(accounts[1]);

        const result = await assetTrailContract.createPaymentInstance(testPaymentInstanceIDs[1], testPaymentDefinitionIDs[1], accounts[1], 1);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.paymentInstanceID, testPaymentInstanceIDs[1]);
        assert.equal(logArgs.paymentDefinitionID, testPaymentDefinitionIDs[1]);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.member, accounts[1]);
        assert.equal(logArgs.amount, 1);
        assert(logArgs.timestamp.toNumber() > 0);

        const balanceAccount0After = await payment.balanceOf(accounts[0]);
        const balanceAccount1After = await payment.balanceOf(accounts[1]);
        assert(balanceAccount0Before.toNumber() === balanceAccount0After.toNumber() + 1);
        assert(balanceAccount1Before.toNumber() === balanceAccount1After.toNumber() - 1);
      });

    });

    describe('Asset properties', () => {

      it('setAssetInstanceProperty should raise an error if the key is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.setAssetInstanceProperty(testAssetDefinitionIDs[0], testAssetInstanceIDs[0], '', testAssetPropertyValues[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid key'));
      });

      it('setAssetInstanceProperty should set an asset property and emit the corresponding event', async () => {
        const result = await assetTrailContract.setAssetInstanceProperty(testAssetDefinitionIDs[0], testAssetInstanceIDs[0], testAssetPropertyKeys[0], testAssetPropertyValues[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, testAssetDefinitionIDs[0]);
        assert.equal(logArgs.assetInstanceID, testAssetInstanceIDs[0]);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.key, testAssetPropertyKeys[0]);
        assert.equal(logArgs.value, testAssetPropertyValues[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

  });

});