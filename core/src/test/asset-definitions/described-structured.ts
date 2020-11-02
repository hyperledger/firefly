import { app, mockEventStreamWebSocket, sampleContentSchema, sampleDescriptionSchema } from '../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IEventAssetDefinitionCreated } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Asset definitions: described - structured', async () => {

  describe('Create public asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io' })
        .post('/api/v0/add')
        .reply(200, { Hash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'Described - structured - public',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: false,
          descriptionSchema: sampleDescriptionSchema,
          contentSchema: sampleContentSchema
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Described - structured - public');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleDescriptionSchema);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Described - structured - public');
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
        assetDefinitionID: '6',
        author: '0x0000000000000000000000000000000000000001',
        name: 'Described - structured - public',
        descriptionSchemaHash: '0xaea64aa86186d5740e8603cb85fd439f339b6e538551d88cf8305bc94271d45f',
        contentSchemaHash: '0xf7b1df6546ec552e2e5a33aec9f16eace7239e3b719105a86a1566683bfd69b2',
        isContentPrivate: false,
        timestamp: '7'
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Described - structured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, 6);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Described - structured - public');
      assert.strictEqual(assetDefinition.timestamp, 7);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/6')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Create private asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io' })
        .post('/api/v0/add')
        .reply(200, { Hash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'Described - structured - private',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: true,
          descriptionSchema: sampleDescriptionSchema,
          contentSchema: sampleContentSchema
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Described - structured - private');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Described - structured - private');
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
        assetDefinitionID: '7',
        author: '0x0000000000000000000000000000000000000001',
        name: 'Described - structured - private',
        descriptionSchemaHash: '0xaea64aa86186d5740e8603cb85fd439f339b6e538551d88cf8305bc94271d45f',
        contentSchemaHash: '0xf7b1df6546ec552e2e5a33aec9f16eace7239e3b719105a86a1566683bfd69b2',
        isContentPrivate: true,
        timestamp: '8'
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'Described - structured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, 7);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleContentSchema);
      assert.strictEqual(assetDefinition.name, 'Described - structured - private');
      assert.strictEqual(assetDefinition.timestamp, 8);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/7')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });


  });

});
