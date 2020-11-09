import { app, mockEventStreamWebSocket } from '../../../common';
import { testDescription, testContent } from '../../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition, IEventAssetInstanceCreated, IDBAssetInstance } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';

describe('Assets: unauthored - public - described - structured', async () => {

  const assetDefinitionID = 'cb289f68-5eaf-46d2-9be1-3dcfdde76885';
  const timestamp = utils.getTimestamp();

  describe('Asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${testDescription.schema.ipfsMultiHash}`)
      .reply(200, testDescription.schema.object)
      .get(`/ipfs/${testContent.schema.ipfsMultiHash}`)
      .reply(200, testContent.schema.object);

      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - public - described - structured',
        descriptionSchemaHash: testDescription.schema.ipfsSha256,
        contentSchemaHash: testContent.schema.ipfsSha256,
        isContentPrivate: false,
        isContentUnique: true,
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_STRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - public - described - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.name, 'unauthored - public - described - structured');
      assert.strictEqual(assetDefinition.timestamp, timestamp);

      const getAssetDefinitionResponse = await request(app)
      .get(`/api/v1/assets/definitions/${assetDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Asset instances', async () => {

    const assetInstanceID = '9e8acd3b-4067-443f-ad6e-739976bc63ee';

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${testDescription.sample.ipfsMultiHash}`)
      .reply(200, testDescription.sample.object)
      .get(`/ipfs/${testContent.sample.ipfsMultiHash}`)
      .reply(200, testContent.sample.object)

      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetInstanceCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000002',
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        descriptionHash: testDescription.sample.ipfsSha256,
        contentHash: testContent.sample.ipfsMultiHash,
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_ASSET_INSTANCE_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset instance is confirmed', async () => {
      const getAssetInstancesResponse = await request(app)
        .get('/api/v1/assets/instances')
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.descriptionHash, testDescription.sample.ipfsSha256);
      assert.deepStrictEqual(assetInstance.description, testDescription.sample.object);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.ipfsMultiHash);
      assert.deepStrictEqual(assetInstance.content, testContent.sample.object);
      assert.strictEqual(assetInstance.confirmed, true);
      assert.strictEqual(typeof assetInstance.timestamp, 'number');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/instances/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);
    });

  });

});
