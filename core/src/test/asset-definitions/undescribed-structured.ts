import { app, mockEventStreamWebSocket, sampleContentSchema } from '../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IEventAssetDefinitionCreated } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Asset definitions: undescribed - structured', async () => {

  describe('Create public asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'QmVQgessb3MwJ9h7hRmgXFYzhNDPfj9ot5SgxWvLjMLgoN' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'Undescribed - structured - public',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: false,
          contentSchema: sampleContentSchema
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Undescribed - structured - public');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Undescribed - structured - public');
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
        assetDefinitionID: '2',
        author: '0x0000000000000000000000000000000000000001',
        name: 'Undescribed - structured - public',
        isContentPrivate: false,
        timestamp: '3'
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.UNSTRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Undescribed - structured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, 2);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Undescribed - structured - public');
      assert.strictEqual(assetDefinition.timestamp, 3);
    });

  });

  describe('Create private asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'QmVQgessb3MwJ9h7hRmgXFYzhNDPfj9ot5SgxWvLjMLgoN' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'Undescribed - structured - private',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: true,
          contentSchema: sampleContentSchema
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Undescribed - structured - private');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Undescribed - structured - private');
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
        assetDefinitionID: '2',
        author: '0x0000000000000000000000000000000000000001',
        name: 'Undescribed - structured - private',
        isContentPrivate: true,
        timestamp: '3'
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.UNSTRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Undescribed - structured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, 2);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Undescribed - structured - private');
      assert.strictEqual(assetDefinition.timestamp, 3);
    });

  });

});
