import { app, mockEventStreamWebSocket } from '../../../common';
import { testContent } from '../../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition, IEventAssetInstanceCreated, IDBAssetInstance } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';

describe('Assets: unauthored - private - structured', async () => {

  const assetDefinitionID = '02d7cbbf-aa9e-4724-a645-86edf4572ff3';
  const timestamp = utils.getTimestamp();

  describe('Asset definition', async () => {


    nock('https://ipfs.kaleido.io')
    .get(`/ipfs/${testContent.schema.ipfsMultiHash}`)
    .reply(200, testContent.schema.object);

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - private - structured',
        contentSchemaHash: testContent.schema.ipfsSha256,
        isContentPrivate: true,
        timestamp: timestamp.toString()
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - private - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.name, 'unauthored - private - structured');
      assert.strictEqual(assetDefinition.timestamp, timestamp);

      const getAssetDefinitionResponse = await request(app)
      .get(`/api/v1/assets/definitions/${assetDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Asset instances', async () => {

    const assetInstanceID = '9d53cce2-0879-4aaa-9b82-cb30eb6331be';

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {

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
        contentHash: testContent.sample.docExchangeSha256,
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
      assert.strictEqual(assetInstance.contentHash, testContent.sample.docExchangeSha256);
      assert.deepStrictEqual(assetInstance.content, undefined);
      assert.strictEqual(assetInstance.confirmed, true);
      assert.strictEqual(typeof assetInstance.timestamp, 'number');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/instances/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);
    });

  });

});
