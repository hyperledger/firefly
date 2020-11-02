import { app, mockEventStreamWebSocket, sampleSchemas } from '../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IEventAssetDefinitionCreated } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Asset definitions: authored - undescribed - structured', async () => {

  describe('Create public asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - undescribed - structured - public',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: false,
          contentSchema: sampleSchemas.assetContent.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - undescribed - structured - public');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.assetContent.object);
      assert.strictEqual(assetDefinition.name, 'authored - undescribed - structured - public');
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
        name: 'authored - undescribed - structured - public',
        contentSchemaHash: sampleSchemas.assetContent.sha256,
        isContentPrivate: false,
        timestamp: '3'
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.STRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - undescribed - structured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, 2);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.assetContent.object);
      assert.strictEqual(assetDefinition.name, 'authored - undescribed - structured - public');
      assert.strictEqual(assetDefinition.timestamp, 3);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/2')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Create private asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - undescribed - structured - private',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: true,
          contentSchema: sampleSchemas.assetContent.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - undescribed - structured - private');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.assetContent.object);
      assert.strictEqual(assetDefinition.name, 'authored - undescribed - structured - private');
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
        assetDefinitionID: '3',
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - undescribed - structured - private',
        contentSchemaHash: sampleSchemas.assetContent.sha256,
        isContentPrivate: true,
        timestamp: '3'
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.STRUCTURED_ASSET_DEFINITION_CREATED,
        data
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - undescribed - structured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, 3);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.assetContent.object);
      assert.strictEqual(assetDefinition.name, 'authored - undescribed - structured - private');
      assert.strictEqual(assetDefinition.timestamp, 3);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/3')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });


  });

});
