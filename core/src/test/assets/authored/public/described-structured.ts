import { app, mockEventStreamWebSocket } from '../../../common';
import { testDescription, testContent } from '../../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IDBAssetInstance, IEventAssetDefinitionCreated, IEventAssetInstanceCreated } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';

describe('Assets: authored - public - described - structured', async () => {

  let assetDefinitionID: string;
  const timestamp = utils.getTimestamp();

  describe('Create asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: testDescription.schema.ipfsMultiHash })
        .post('/api/v0/add')
        .reply(200, { Hash: testContent.schema.ipfsMultiHash });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - public - described - structured',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: false,
          isContentUnique: true,
          descriptionSchema: testDescription.schema.object,
          contentSchema: testContent.schema.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      assetDefinitionID = result.body.assetDefinitionID;

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - public - described - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.name, 'authored - public - described - structured');
      assert.strictEqual(typeof assetDefinition.timestamp, 'number');
    });

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - public - described - structured',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - public - described - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.name, 'authored - public - described - structured');
      assert.strictEqual(assetDefinition.timestamp, timestamp);

      const getAssetDefinitionResponse = await request(app)
        .get(`/api/v1/assets/definitions/${assetDefinitionID}`)
        .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Asset instances', async () => {

    let assetInstanceID: string;

    it('Checks that an asset instance can be created', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedAssetInstance?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: testDescription.sample.ipfsMultiHash })
        .post('/api/v0/add')
        .reply(200, { Hash: testContent.sample.ipfsMultiHash })

      const result = await request(app)
        .post('/api/v1/assets/instances')
        .send({
          assetDefinitionID,
          author: '0x0000000000000000000000000000000000000001',
          description: testDescription.sample.object,
          content: testContent.sample.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      assetInstanceID = result.body.assetInstanceID;

      const getAssetInstancesResponse = await request(app)
        .get('/api/v1/assets/instances')
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.descriptionHash, testDescription.sample.ipfsSha256);
      assert.deepStrictEqual(assetInstance.description, testDescription.sample.object);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.ipfsSha256);
      assert.deepStrictEqual(assetInstance.content, testContent.sample.object);
      assert.strictEqual(assetInstance.confirmed, false);
      assert.strictEqual(typeof assetInstance.timestamp, 'number');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/instances/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);

    });

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetInstanceCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000001',
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        descriptionHash: testDescription.sample.ipfsSha256,
        contentHash: testContent.sample.ipfsSha256,
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
      assert.strictEqual(assetInstance.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.descriptionHash, testDescription.sample.ipfsSha256);
      assert.deepStrictEqual(assetInstance.description, testDescription.sample.object);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.ipfsSha256);
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
