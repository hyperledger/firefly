const AssetTrail = artifacts.require('AssetTrail');
const AssetTrailImplV1 = artifacts.require('AssetTrailImplV1');
const crypto = require('crypto');
const ABI = require('./abi.json');

const testMemberNames = [
  'member-1',
  'member-2',
  'member-3'
];

const testDestinations = [
  'kld://documentstore-destination-1',
  'kld://documentstore-destination-2',
  'kld://documentstore-destination-3'
];

const testKeyHashes = [
  '0x6ab9f1eb8f7d3388f4f9d586f66e99fd54080df2c446f0e58668b09c08a16dd0',
  '0x015f7e6bc5aeaf483724089e9252cc13b50951a6b69412522765cff4d780306e'
];

const testContentHashes = [
  '0xd0f631ca1ddba8db3bcfcb9e057cdc98d0379f1bee00e75a545147a27dadd982',
  '0x9c0abe51c6e6655d81de2d044d4fb194931f058c0426c67c7285d8f5657ed64a'
];

const testAttachmentHashes = [
  '0xf55ff16f66f43360266b95db6f8fec01d76031054306ae4a4b380598f6cfd114',
  '0x2c3a4249d77070058649dbd822dcaf7957586fce428cfb2ca88b94741eda8b07',
  '0xf46dd28a5499d8efef0b8fb8ee1ec1c5a5e407c9381741d576ba8deb4f59ec3f'
];

const testAttachmentDescriptions = [
  'description-1',
  'description-2'
];

const testAttachmentCosts = [
  1,
  2
];

const testMetadataKeys = [
  'key-1',
  'key-2'
];

const testMetadataValues = [
  'value-1',
  'value-2'
];

contract('AssetTrail tests', (accounts) => {
  let implv1, mainContract;
  let proxyAdmin, user1, user2, user3, user4;
  const TEST_HASH = '0x' + crypto.createHash('sha256').update('Some fact').digest('hex');

  function submit(data, user = user1) {
    return web3.eth.sendTransaction({
      to: mainContract.address,
      from: user,
      data,
      gas: 1000000
    });
  }

  before(() => {
    proxyAdmin = accounts[0];
    user1 = accounts[1];
    user2 = accounts[2];
    user3 = accounts[3];
    user4 = accounts[4];
  });

  describe('constructor tests', () => {
    it('deploys impl v1', async () => {
      implv1 = await AssetTrailImplV1.new();
    });

    it('deploys main contract', async () => {
      const data = web3.eth.abi.encodeFunctionCall(
        {
          "type": "function",
          "name": "initialize",
          "inputs": []
        },
        []
      );

      mainContract = await AssetTrail.new(implv1.address, proxyAdmin, data, { from: proxyAdmin });
    });
  });

  describe('Members', () => {

    it('registerMember show raise an error if the name is empty', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_registerMember, ['', testDestinations[0]]);
      try {
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Invalid member name'));
    });

    it('registerMember show raise an error if the destination is empty', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_registerMember, [testMemberNames[0], '']);
      try {
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Invalid member destination'));
    });

    it('registerMember should register a member and emit the corresponding event (member 1)', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_registerMember, [testMemberNames[0], testDestinations[0]]);
      const result = await submit(data);
      let event = web3.eth.abi.decodeLog(ABI.event_MemberRegistered, result.logs[0].data, result.logs[0].topics.slice(1));
      expect(event.member).to.equal(user1);
      expect(event.name).to.equal(testMemberNames[0]);
      expect(event.destination).to.equal(testDestinations[0]);
      expect(parseInt(event.timestamp) > 0).to.equal(true);
    });

    it('registerMember should register a member and emit the corresponding event (member 2)', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_registerMember, [testMemberNames[1], testDestinations[1]]);
      const result = await submit(data, user2);
      let event = web3.eth.abi.decodeLog(ABI.event_MemberRegistered, result.logs[0].data, result.logs[0].topics.slice(1));
      expect(event.member).to.equal(user2);
      expect(event.name).to.equal(testMemberNames[1]);
      expect(event.destination).to.equal(testDestinations[1]);
      expect(parseInt(event.timestamp) > 0).to.equal(true);
    });

    it('registerMember should register a member and emit the corresponding event (member 3)', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_registerMember, [testMemberNames[2], testDestinations[2]]);
      const result = await submit(data, user3);
      let event = web3.eth.abi.decodeLog(ABI.event_MemberRegistered, result.logs[0].data, result.logs[0].topics.slice(1));
      expect(event.member).to.equal(user3);
      expect(event.name).to.equal(testMemberNames[2]);
      expect(event.destination).to.equal(testDestinations[2]);
      expect(parseInt(event.timestamp) > 0).to.equal(true);
    });

    it('registerMember should raise an error when registering the same member more than once', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_registerMember, [testMemberNames[0], testDestinations[0]]);
      try {
        const result = await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Member already registered'));
    });

  });

  describe('Entries', () => {

    it('createEntry should raise an error if the sender is not a registered member', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_createEntry, [testKeyHashes[0], testContentHashes[0]]);
      try {
        const result = await submit(data, user4);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Member must be registered'));
    });

    it('createEntry should create an entry and emit the corresponding event', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_createEntry, [testKeyHashes[0], testContentHashes[0]]);
      const result = await submit(data);
      let event = web3.eth.abi.decodeLog(ABI.event_EntryCreated, result.logs[0].data, result.logs[0].topics.slice(1));
      assert.equal(event.keyHash, testKeyHashes[0]);
      assert.equal(event.author, user1);
      assert.equal(event.contentHash, testContentHashes[0]);
      assert(parseInt(event.timestamp) > 0);
    });

    it('createEntry should raise an error when attempting to create the same entry more than once', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_createEntry, [testKeyHashes[0], testContentHashes[0]]);
      try {
        const result = await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Entry already exists'));
    });

  });

  describe('Subscriptions', () => {

    it('addEntrySubscriber should raise an error if the member is not registered', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[0], user2]);
      try {
        const result = await submit(data, user4);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Member must be registered'));
    });

    it('addEntrySubscriber should raise an error if the subscriber is not registered', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[0], user4]);
      try {
        const result = await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Subscriber must be registered'));
    });

    it('addEntrySubscriber should raise an error if the entry does not exist', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[1], user2]);
      try {
        const result = await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Entry not found'));
    });

    it('addEntrySubscriber should raise an error if the sender is not the entry author', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[0], user1]);
      try {
        const result = await submit(data, user2);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Must be entry author to add subsribers'));
    });

    it('addEntrySubscriber should raise an error if the subscriber is the entry author', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[0], user1]);
      try {
        const result = await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Cannot subscribe to authored entry'));
    });

    it('addEntrySubscriber should add a subscriber, transfer tokens and emit the corresponding event', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[0], user2]);
      const result = await submit(data);
      let event = web3.eth.abi.decodeLog(ABI.event_SubscriptionAdded, result.logs[0].data, result.logs[0].topics.slice(1));
      assert.equal(event.keyHash, testKeyHashes[0]);
      assert.equal(event.subscriber, user2);
      assert(parseInt(event.timestamp) > 0);
    });

    it('addEntrySubscriber should raise an error when attempting to subscribe the same member to an entry more than once', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addEntrySubscriber, [testKeyHashes[0], user2]);
      try {
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Already subscribed'));
    });

  });

  describe('Attachments', () => {

    it('addAttachment should raise an error if the description is empty', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addAttachment, [testKeyHashes[1], testAttachmentHashes[0], '']);
      try {
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Description cannot be empty'));
    });

    it('addAttachment should raise an error if the entry does not exist', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addAttachment, [testKeyHashes[1], testAttachmentHashes[0], testAttachmentDescriptions[0]]);
      try {
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Entry not found'));
    });

    it('addAttachment should raise an error if the sender is neither the author nor a subscriber', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addAttachment, [testKeyHashes[0], testAttachmentHashes[0], testAttachmentDescriptions[0]]);
      try {
        await submit(data, user3);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Must be entry author or subscriber to add attachment'));
    });

    it('addAttachment should add an attachment and emit the corresponding event (entry author)', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addAttachment, [testKeyHashes[0], testAttachmentHashes[0], testAttachmentDescriptions[0]]);
      const result = await submit(data);
      let event = web3.eth.abi.decodeLog(ABI.event_AttachmentAdded, result.logs[0].data, result.logs[0].topics.slice(1));
      assert.equal(event.keyHash, testKeyHashes[0]);
      assert.equal(event.attachmentHash, testAttachmentHashes[0]);
      assert.equal(event.author, user1);
      assert.equal(event.description, testAttachmentDescriptions[0]);
      assert(parseInt(event.timestamp) > 0);
    });

    it('addAttachment should add an attachment and emit the corresponding event (entry subscriber)', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addAttachment, [testKeyHashes[0], testAttachmentHashes[1], testAttachmentDescriptions[1]]);
      const result = await submit(data, user2);
      let event = web3.eth.abi.decodeLog(ABI.event_AttachmentAdded, result.logs[0].data, result.logs[0].topics.slice(1));
      assert.equal(event.keyHash, testKeyHashes[0]);
      assert.equal(event.attachmentHash, testAttachmentHashes[1]);
      assert.equal(event.author, user2);
      assert.equal(event.description, testAttachmentDescriptions[1]);
      assert(parseInt(event.timestamp) > 0);
    });

    it('addAttachment should raise an error if an attachment is added more than once', async () => {
      let exceptionMessage;
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addAttachment, [testKeyHashes[0], testAttachmentHashes[0], testAttachmentDescriptions[0]]);
      try {
        await submit(data, user2);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Attachment already exists'));
    });

  });

  describe('Metadata', () => {

    it('addMetadata should raise an error if the key is empty', async () => {
      let exceptionMessage;
      try {
        const data = web3.eth.abi.encodeFunctionCall(ABI.func_addMetadata, [testKeyHashes[0], '', testMetadataValues[0]]);
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Key cannot be empty'));
    });

    it('addMetadata should raise an error if the value is empty', async () => {
      let exceptionMessage;
      try {
        const data = web3.eth.abi.encodeFunctionCall(ABI.func_addMetadata, [testKeyHashes[0], testMetadataKeys[0], '']);
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Value cannot be empty'));
    });

    it('addMetadata should raise an error if the entry does not exist', async () => {
      let exceptionMessage;
      try {
        const data = web3.eth.abi.encodeFunctionCall(ABI.func_addMetadata, [testKeyHashes[1], testMetadataKeys[0], testMetadataValues[0]]);
        await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Entry not found'));
    });

    it('addMetadata should raise an error if the sender is neither the author nor a subscriber', async () => {
      let exceptionMessage;
      try {
        const data = web3.eth.abi.encodeFunctionCall(ABI.func_addMetadata, [testKeyHashes[0], testMetadataKeys[0], testMetadataValues[0]]);
        await submit(data, user3);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Must be entry author or subscriber to add metadata'));
    });

    it('addMetadata should add metadata and emit the corresponding event (attachment author)', async () => {
      const data = web3.eth.abi.encodeFunctionCall(ABI.func_addMetadata, [testKeyHashes[0], testMetadataKeys[0], testMetadataValues[0]]);
      const result = await submit(data);
      let event = web3.eth.abi.decodeLog(ABI.event_MetadataAdded, result.logs[0].data, result.logs[0].topics.slice(1));
      assert.equal(event.keyHash, testKeyHashes[0]);
      assert.equal(event.author, user1);
      assert.equal(event.key, testMetadataKeys[0]);
      assert.equal(event.value, testMetadataValues[0]);
      assert(parseInt(event.timestamp) > 0);
    });

    it('addMetadata should raise an error if a member adds the same metadata more than once', async () => {
      let exceptionMessage;
      try {
        const data = web3.eth.abi.encodeFunctionCall(ABI.func_addMetadata, [testKeyHashes[0], testMetadataKeys[0], testMetadataValues[0]]);
        const result = await submit(data);
      } catch (err) {
        exceptionMessage = err.message;
      }
      assert(exceptionMessage.includes('Metadata already set'));
    });

  });
});