import { app, mockEventStreamWebSocket, sampleSchemas } from '../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IEventAssetDefinitionCreated } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Asset definitions: authored - described - unstructured', async () => {

  describe('Create public asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedUnstructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - described - unstructured - public',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: false,
          descriptionSchema: sampleSchemas.assetDescription.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - described - unstructured - public');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.assetDescription.object);
      assert.strictEqual(assetDefinition.name, 'authored - described - unstructured - public');
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
        assetDefinitionID: '4',
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - described - unstructured - public',
        descriptionSchemaHash: sampleSchemas.assetDescription.sha256,
        isContentPrivate: false,
        timestamp: '5'
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_UNSTRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - described - unstructured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, 4);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.assetDescription.object);
      assert.strictEqual(assetDefinition.name, 'authored - described - unstructured - public');
      assert.strictEqual(assetDefinition.timestamp, 5);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/4')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Create private asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedUnstructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - described - unstructured - private',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: true,
          descriptionSchema: sampleSchemas.assetDescription.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - described - unstructured - private');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.assetDescription.object);
      assert.strictEqual(assetDefinition.name, 'authored - described - unstructured - private');
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
        assetDefinitionID: '5',
        author: '0x0000000000000000000000000000000000000001',
        descriptionSchemaHash: sampleSchemas.assetDescription.sha256,
        name: 'authored - described - unstructured - private',
        isContentPrivate: true,
        timestamp: '6'
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_UNSTRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - described - unstructured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, 5);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.assetDescription.object);
      assert.strictEqual(assetDefinition.name, 'authored - described - unstructured - private');
      assert.strictEqual(assetDefinition.timestamp, 6);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/5')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });


  });

});
