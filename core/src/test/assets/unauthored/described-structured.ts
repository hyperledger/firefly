import { app, getNextAssetDefinitionID, mockEventStreamWebSocket, sampleSchemas } from '../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../../lib/interfaces';
import * as utils from '../../../lib/utils';

let publicAssetDefinitionID = getNextAssetDefinitionID();
let privateAssetDefinitionID = getNextAssetDefinitionID();

describe('Asset definitions: unauthored - described - structured', async () => {

  const timestamp = utils.getTimestamp();

  describe('Public asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${sampleSchemas.description.multiHash}`)
      .reply(200, sampleSchemas.description.object)
      .get(`/ipfs/${sampleSchemas.content.multiHash}`)
      .reply(200, sampleSchemas.content.object);

      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: publicAssetDefinitionID.toString(),
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - described - structured - public',
        descriptionSchemaHash: sampleSchemas.description.sha256,
        contentSchemaHash: sampleSchemas.content.sha256,
        isContentPrivate: false,
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - described - structured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, publicAssetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.description.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.content.object);
      assert.strictEqual(assetDefinition.name, 'unauthored - described - structured - public');
      assert.strictEqual(assetDefinition.timestamp, timestamp);

      const getAssetDefinitionResponse = await request(app)
      .get(`/api/v1/assets/definitions/${publicAssetDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Private asset definition', async () => {

    const timestamp = utils.getTimestamp();

    nock('https://ipfs.kaleido.io')
    .get(`/ipfs/${sampleSchemas.description.multiHash}`)
    .reply(200, sampleSchemas.description.object)
    .get(`/ipfs/${sampleSchemas.content.multiHash}`)
    .reply(200, sampleSchemas.content.object);

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: privateAssetDefinitionID.toString(),
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - described - structured - private',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - described - structured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, privateAssetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, sampleSchemas.description.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, sampleSchemas.content.object);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.name, 'unauthored - described - structured - private');
      assert.strictEqual(assetDefinition.timestamp, timestamp);

      const getAssetDefinitionResponse = await request(app)
      .get(`/api/v1/assets/definitions/${privateAssetDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

});
