import { app, mockEventStreamWebSocket } from '../../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';

describe('Assets: authored - private - unstructured', async () => {

  let assetDefinitionID: string;

  describe('Create private asset definition', async () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createUnstructuredAssetDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200, { id: 'my-receipt-id' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: 'authored - private - unstructured',
          author: '0x0000000000000000000000000000000000000001',
          isContentPrivate: true,
          isContentUnique: true,
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      assetDefinitionID = result.body.assetDefinitionID;

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - private - unstructured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.strictEqual(assetDefinition.name, 'authored - private - unstructured');
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
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - private - unstructured',
        isContentPrivate: true,
        isContentUnique: true,
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.UNSTRUCTURED_ASSET_DEFINITION_CREATED,
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - private - unstructured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.strictEqual(assetDefinition.name, 'authored - private - unstructured');
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
