import { app, getNextAssetDefinitionID, mockEventStreamWebSocket, sampleSchemas } from '../../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IEventAssetDefinitionCreated } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';

let assetDefinitionID = getNextAssetDefinitionID();

describe('Assets: authored - private - described - structured', async () => {

  describe('Create asset definition', () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedStructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: sampleSchemas.description.multiHash })
        .post('/api/v0/add')
        .reply(200, { Hash: sampleSchemas.content.multiHash });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - private - described - structured',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: true,
          descriptionSchema: sampleSchemas.description.object,
          contentSchema: sampleSchemas.content.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - private - described - structured');
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, false);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.description.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.content.object);
      assert.strictEqual(assetDefinition.name, 'authored - private - described - structured');
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
        assetDefinitionID: assetDefinitionID.toString(),
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - private - described - structured',
        descriptionSchemaHash: sampleSchemas.description.sha256,
        contentSchemaHash: sampleSchemas.content.sha256,
        isContentPrivate: true,
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - private - described - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.description.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.content.object);
      assert.strictEqual(assetDefinition.name, 'authored - private - described - structured');
      assert.strictEqual(assetDefinition.timestamp, timestamp);

      const getAssetDefinitionResponse = await request(app)
      .get(`/api/v1/assets/definitions/${assetDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Asset instances', () => {

    it('Attempting to create an asset instance without a description should raise an error', async () => {
      const result = await request(app)
        .post('/api/v1/assets/instances')
        .send({
          assetDefinitionID,
          author: '0x0000000000000000000000000000000000000001',
          content: sampleSchemas.content.sample
        })
        .expect(400);
      assert.deepStrictEqual(result.body, { error: 'Missing asset definition' });
    });

    it('Attempting to create an asset instance with an invalid description should raise an error', async () => {
      const result = await request(app)
        .post('/api/v1/assets/instances')
        .send({
          assetDefinitionID,
          author: '0x0000000000000000000000000000000000000001',
          description: {},
          content: sampleSchemas.content.sample
        })
        .expect(400);
      assert.deepStrictEqual(result.body, { error: 'Description does not conform to asset definition schema' });
    });

    // it('Attempting to create an asset instance with invalid content should raise an error', async () => {
    //   const result = await request(app)
    //     .post('/api/v1/assets/instances')
    //     .send({
    //       assetDefinitionID,
    //       author: '0x0000000000000000000000000000000000000001',
    //       description: sampleSchemas.description.sample,
    //       content: {}
    //     })
    //     .expect(400);
    //   assert.deepStrictEqual(result.body, { error: 'Content does not conform to asset definition schema' });
    // });

  });

});
