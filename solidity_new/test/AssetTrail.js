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

const testAssetDefinitionNames = [
  'My described structured asset',
  'My described unstructured asset',
  'My structured asset',
  'My unstructured asset',
];

const testDescriptionSchemaHashes = [
  '0x6ab9f1eb8f7d3388f4f9d586f66e99fd54080df2c446f0e58668b09c08a16dd0'
];

const testContentSchemaHashes = [
  '0xd0f631ca1ddba8db3bcfcb9e057cdc98d0379f1bee00e75a545147a27dadd982'
];

// Asset instance constants

const testDescriptionHashes = [
  '0x27a84712e4b22c415fc544d55cdee82327a829f96d03329457f76ebf9af4dcaa',
  '0x13609d74cc8ea07555856a54ba51b01f831af4af89bd39847babe5bf6cb665df'
];

const testContentHashes = [
  '0x4fb431659a5b45f4e7b1a69bacb4101a11b82777de6857c3e40d7fe217307285',
  '0x1e335362351e60f908f58ac674f8c0967dca8af41c3b696066b49810d399d795'
];

// Payment definition constants

const testPaymentDefinitionNames = [
  'My subscriptions'
];

const testPaymentSchemas = [
  '0x6dee8ee9d5a7743e2a86e03e652f072f520d0955d6c7551ba4c85f71011d0896'
];

const testPaymentAmounts = [
  10
];

// Payment instance constants

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

    describe('Status', () => {

      it('Initial', async() => {
        const result = await assetTrailContract.getStatus();
        assert.equal(result.totalAssetDefinitions, 0);
        assert.equal(result.totalPaymentDefinitionsc, 0);
        assert.equal(result.totalAssetInstances, 0);
        assert.equal(result.totalPaymentInstances, 0);
      });

  });

    describe('Members', () => {

      it('registerMember should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.registerMember('', testApp2AppDestinations[0], testDocExchangeDestinations[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('registerMember should register a member and emit the corresponding event (member 1)', async () => {
        const result = await assetTrailContract.registerMember(testMemberNames[0], testApp2AppDestinations[0], testDocExchangeDestinations[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.member, accounts[0]);
        assert.equal(logArgs.name, testMemberNames[0]);
        assert.equal(logArgs.app2appDestination, testApp2AppDestinations[0]);
        assert.equal(logArgs.docExchangeDestination, testDocExchangeDestinations[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('registerMember should register a member and emit the corresponding event (member 2)', async () => {
        const result = await assetTrailContract.registerMember(testMemberNames[1], testApp2AppDestinations[1], testDocExchangeDestinations[1], { from: accounts[1] });
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.member, accounts[1]);
        assert.equal(logArgs.name, testMemberNames[1]);
        assert.equal(logArgs.app2appDestination, testApp2AppDestinations[1]);
        assert.equal(logArgs.docExchangeDestination, testDocExchangeDestinations[1]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('registerMember should allow members to update their name and destinations (member 2)', async () => {
        const result = await assetTrailContract.registerMember(testMemberNames[2], testApp2AppDestinations[2], testDocExchangeDestinations[2], { from: accounts[1] });
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.member, accounts[1]);
        assert.equal(logArgs.name, testMemberNames[2]);
        assert.equal(logArgs.app2appDestination, testApp2AppDestinations[2]);
        assert.equal(logArgs.docExchangeDestination, testDocExchangeDestinations[2]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Described structured asset definitions', () => {

      it('createDescribedStructuredAssetDefinition should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedStructuredAssetDefinition(testAssetDefinitionNames[0], true, testDescriptionSchemaHashes[0], testContentSchemaHashes[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createDescribedStructuredAssetDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedStructuredAssetDefinition('', true, testDescriptionSchemaHashes[0], testContentSchemaHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createDescribedStructuredAssetDefinition should create a new described structured asset definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createDescribedStructuredAssetDefinition(testAssetDefinitionNames[0], true, testDescriptionSchemaHashes[0], testContentSchemaHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, 0);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.name, testAssetDefinitionNames[0]);
        assert.equal(logArgs.isContentPrivate, true);
        assert.equal(logArgs.descriptionSchemaHash, testDescriptionSchemaHashes[0]);
        assert.equal(logArgs.contentSchemaHash, testContentSchemaHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('createDescribedStructuredAssetDefinition should raise an error when there is a name conflict', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedStructuredAssetDefinition(testAssetDefinitionNames[0], true, testDescriptionSchemaHashes[0], testContentSchemaHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Asset definition name conflict'));
      });

    });

    describe('Described unstructured asset definitions', () => {

      it('createDescribedUnstructuredAssetDefinition should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedUnstructuredAssetDefinition(testAssetDefinitionNames[1], true, testDescriptionSchemaHashes[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createDescribedUnstructuredAssetDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedUnstructuredAssetDefinition('', true, testDescriptionSchemaHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createDescribedUnstructuredAssetDefinition should create a new described unstructured asset definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createDescribedUnstructuredAssetDefinition(testAssetDefinitionNames[1], true, testDescriptionSchemaHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, 1);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.name, testAssetDefinitionNames[1]);
        assert.equal(logArgs.isContentPrivate, true);
        assert.equal(logArgs.descriptionSchemaHash, testDescriptionSchemaHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('createDescribedUnstructuredAssetDefinition should raise an error when there is a name conflict', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedUnstructuredAssetDefinition(testAssetDefinitionNames[1], true, testDescriptionSchemaHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Asset definition name conflict'));
      });

    });

    describe('Structured asset definitions', () => {

      it('createStructuredAssetDefinition should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createStructuredAssetDefinition(testAssetDefinitionNames[2], true, testContentSchemaHashes[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createStructuredAssetDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createStructuredAssetDefinition('', true, testContentSchemaHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createStructuredAssetDefinition should create a new structured asset definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createStructuredAssetDefinition(testAssetDefinitionNames[2], true, testContentSchemaHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, 2);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.name, testAssetDefinitionNames[2]);
        assert.equal(logArgs.isContentPrivate, true);
        assert.equal(logArgs.contentSchemaHash, testContentSchemaHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('createStructuredAssetDefinition should raise an error when there is a name conflict', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createStructuredAssetDefinition(testAssetDefinitionNames[2], true, testContentSchemaHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Asset definition name conflict'));
      });

    });

    describe('Unatructured asset definitions', () => {

      it('createUnstructuredAssetDefinition should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createUnstructuredAssetDefinition(testAssetDefinitionNames[2], true, { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createUnstructuredAssetDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createUnstructuredAssetDefinition('', true);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createUnstructuredAssetDefinition should create a new unstructured asset definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createUnstructuredAssetDefinition(testAssetDefinitionNames[3], true);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, 3);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.name, testAssetDefinitionNames[3]);
        assert.equal(logArgs.isContentPrivate, true);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('createUnstructuredAssetDefinition should raise an error when there is a name conflict', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createUnstructuredAssetDefinition(testAssetDefinitionNames[3], true);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Asset definition name conflict'));
      });

    });

    describe('Described asset instances', () => {

      it('createDescribedAssetInstance should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createDescribedAssetInstance(0, testDescriptionHashes[0], testContentHashes[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createDescribedAssetInstance should create a new described asset instance and emit the corresponding event', async () => {
        const result = await assetTrailContract.createDescribedAssetInstance(0, testDescriptionHashes[0], testContentHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, 0);
        assert.equal(logArgs.assetInstanceID, 0);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.descriptionHash, testDescriptionHashes[0]);
        assert.equal(logArgs.contentHash, testContentHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Asset instances', () => {

      it('createAssetInstance should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createAssetInstance(2, testContentHashes[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createAssetInstance should create a new asset instance and emit the corresponding event', async () => {
        const result = await assetTrailContract.createAssetInstance(2, testContentHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetDefinitionID, 2);
        assert.equal(logArgs.assetInstanceID, 1);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.contentHash, testContentHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Payment definitions', () => {

      it('createPaymentDefinition should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentDefinition(testPaymentDefinitionNames[0], testPaymentSchemas[0], testPaymentAmounts[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createPaymentDefinition should raise an error if the name is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentDefinition('', testPaymentSchemas[0], testPaymentAmounts[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid name'));
      });

      it('createPaymentDefinition should raise an error if the payment amount is 0', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentDefinition(testPaymentDefinitionNames[0], testPaymentSchemas[0], 0);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid amount'));
      });

      it('createPaymentDefinition should create a new payment definition and emit the corresponding event', async () => {
        const result = await assetTrailContract.createPaymentDefinition(testPaymentDefinitionNames[0], testPaymentSchemas[0], testPaymentAmounts[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.paymentDefinitionID, 0);
        assert.equal(logArgs.name, testPaymentDefinitionNames[0]);
        assert.equal(logArgs.paymentSchema, testPaymentSchemas[0]);
        assert.equal(logArgs.amount, testPaymentAmounts[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

      it('createPaymentDefinition should raise an error when there is a name conflict', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentDefinition(testPaymentDefinitionNames[0], testPaymentSchemas[0], testPaymentAmounts[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Payment definition name conflict'));
      });

    });

    describe('Payment instances', () => {

      it('createPaymentInstance should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentInstance(0, accounts[1], testPaymentHashes[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('createPaymentInstance should raise an error if the recipient is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentInstance(0, accounts[2], testPaymentHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Recipient must be registered'));
      });

      it('createPaymentInstance should raise an error if author and recipient are the same', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.createPaymentInstance(0, accounts[0], testPaymentHashes[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Author and recipient cannot be the same'));
      });

      it('createPaymentInstance should create a new payment instance and emit the corresponding event', async () => {
        const balanceAccount0Before = await payment.balanceOf(accounts[0]);
        const balanceAccount1Before = await payment.balanceOf(accounts[1]);

        const result = await assetTrailContract.createPaymentInstance(0, accounts[1], testPaymentHashes[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.paymentDefinitionID, 0);
        assert.equal(logArgs.paymentInstanceID, 0);
        assert.equal(logArgs.author, accounts[0]);
        assert.equal(logArgs.recipient, accounts[1]);
        assert.equal(logArgs.paymentHash, testPaymentHashes[0]);
        assert(logArgs.timestamp.toNumber() > 0);

        const balanceAccount0After = await payment.balanceOf(accounts[0]);
        const balanceAccount1After = await payment.balanceOf(accounts[1]);
        assert(balanceAccount0Before.toNumber() === balanceAccount0After.toNumber() + testPaymentAmounts[0]);
        assert(balanceAccount1Before.toNumber() === balanceAccount1After.toNumber() - testPaymentAmounts[0]);
      });

    });

    describe('Asset properties', () => {

      it('setAssetProperty should raise an error if the sender is not a registered member', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.setAssetProperty(0, testAssetPropertyKeys[0], testAssetPropertyValues[0], { from: accounts[2] });
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Member must be registered'));
      });

      it('setAssetProperty should raise an error if the key is empty', async () => {
        let exceptionMessage;
        try {
          await assetTrailContract.setAssetProperty(0, '', testAssetPropertyValues[0]);
        } catch (err) {
          exceptionMessage = err.message;
        }
        assert(exceptionMessage.includes('Invalid key'));
      });

      it('setAssetProperty should set an asset property and emit the corresponding event', async () => {
        const result = await assetTrailContract.setAssetProperty(0, testAssetPropertyKeys[0], testAssetPropertyValues[0]);
        const logArgs = result.logs[0].args;
        assert.equal(logArgs.assetInstanceID, 0);
        assert.equal(logArgs.propertyAuthor, accounts[0]);
        assert.equal(logArgs.key, testAssetPropertyKeys[0]);
        assert.equal(logArgs.value, testAssetPropertyValues[0]);
        assert(logArgs.timestamp.toNumber() > 0);
      });

    });

    describe('Status', () => {

      it('Final', async() => {
        const result = await assetTrailContract.getStatus();
        assert.equal(result.totalAssetDefinitions, 4);
        assert.equal(result.totalPaymentDefinitionsc, 1);
        assert.equal(result.totalAssetInstances, 2);
        assert.equal(result.totalPaymentInstances, 1);
      });

    });

  });

});