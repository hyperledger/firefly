import { app, mockEventStreamWebSocket, sampleSchemas } from '../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Asset definitions: unauthored - undescribed - structured', async () => {

  describe('Public asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${sampleSchemas.assetContent.multiHash}`)
      .reply(200, sampleSchemas.assetContent.object);

      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: '10',
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - undescribed - structured - public',
        contentSchemaHash: sampleSchemas.assetContent.sha256,
        isContentPrivate: false,
        timestamp: '11'
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - undescribed - structured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, 10);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.assetContent.object);
      assert.strictEqual(assetDefinition.name, 'unauthored - undescribed - structured - public');
      assert.strictEqual(assetDefinition.timestamp, 11);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/10')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Private asset definition', async () => {

    nock('https://ipfs.kaleido.io')
    .get(`/ipfs/${sampleSchemas.assetContent.multiHash}`)
    .reply(200, sampleSchemas.assetContent.object);

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: '11',
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - undescribed - structured - private',
        contentSchemaHash: sampleSchemas.assetContent.sha256,
        isContentPrivate: true,
        timestamp: '12'
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - undescribed - structured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, 11);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.name, 'unauthored - undescribed - structured - private');
      assert.strictEqual(assetDefinition.timestamp, 12);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/11')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

});
