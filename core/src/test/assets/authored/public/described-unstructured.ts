import { app, mockEventStreamWebSocket } from '../../../common';
import { testAssetDefinition, testDescription, getUnstructuredAssetDefinition } from '../../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IEventAssetDefinitionCreated } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';

describe('Asset definitions: authored - public - described - unstructured', async () => {

  let assetDefinitionID: string;

  describe('Create asset definition', () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200, { id: 'my-receipt-id' });

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: testDescription.schema.ipfsMultiHash });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - public - described - unstructured',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: false,
          isContentUnique: true,
          descriptionSchema: testDescription.schema.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      assetDefinitionID = result.body.assetDefinitionID;

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - public - described - unstructured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(assetDefinition.name, 'authored - public - described - unstructured');
      assert.strictEqual(assetDefinition.receipt, 'my-receipt-id');
      assert.strictEqual(typeof assetDefinition.submitted, 'number');
    });

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });

      nock('https://ipfs.kaleido.io')
        .get(`/ipfs/${testAssetDefinition.ipfsMultiHash}`)
        .reply(200, getUnstructuredAssetDefinition(assetDefinitionID, 'authored - public - described - unstructured', false));
      const data: IEventAssetDefinitionCreated = {
        author: '0x0000000000000000000000000000000000000001',
        assetDefinitionHash: testAssetDefinition.ipfsSha256,
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.ASSET_DEFINITION_CREATED,
        data,
        blockNumber: '123',
        transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
      }]));
      await eventPromise;
    });

    it('Checks that the asset definition is confirmed', async () => {
      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - public - described - unstructured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(assetDefinition.name, 'authored - public - described - unstructured');
      assert.strictEqual(typeof assetDefinition.submitted, 'number');
      assert.strictEqual(assetDefinition.timestamp, timestamp);
      assert.strictEqual(assetDefinition.receipt, 'my-receipt-id');
      assert.strictEqual(assetDefinition.blockNumber, 123);
      assert.strictEqual(assetDefinition.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getAssetDefinitionResponse = await request(app)
        .get(`/api/v1/assets/definitions/${assetDefinitionID}`)
        .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

});
