import { app, mockEventStreamWebSocket } from '../../../../common';
import { testContent } from '../../../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition, IEventAssetInstanceCreated, IDBAssetInstance } from '../../../../../lib/interfaces';
import * as utils from '../../../../../lib/utils';

export const testUnauthoredPublicStructured = () => {

describe('Assets: unauthored - public - structured', async () => {

  const assetDefinitionID = 'f5847e19-8632-44fb-a742-17b7a1fd8bc5';
  const timestamp = utils.getTimestamp();

  describe('Asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
        .get('/ipfs/QmdptNuREBKs2ruCrfb7EwUJcEXj5oEpmQ5C4MSKht6SmY')
        .reply(200,{
          assetDefinitionID: assetDefinitionID,
          name: 'unauthored - public - structured',
          isContentPrivate: false,
          isContentUnique: true,
          contentSchema: testContent.schema.object
        });

      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        author: '0x0000000000000000000000000000000000000002',
        assetDefinitionHash: '0xe61b05a5914b8b0361d2ae2fa91ac0b76de09a517ef411df049b4af3844b4593',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - public - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.name, 'unauthored - public - structured');
      assert.strictEqual(assetDefinition.timestamp, timestamp);
      assert.strictEqual(assetDefinition.submitted, undefined);
      assert.strictEqual(assetDefinition.receipt, undefined);
      assert.strictEqual(assetDefinition.blockNumber, 123);
      assert.strictEqual(assetDefinition.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getAssetDefinitionResponse = await request(app)
      .get(`/api/v1/assets/definitions/${assetDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Asset instances', async () => {

    const assetInstanceID = '28bc0765-0261-415d-95cf-ea285c91a09c';

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${testContent.sample.ipfsMultiHash}`)
      .reply(200, testContent.sample.object)

      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetInstanceCreated = {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        author: '0x0000000000000000000000000000000000000002',
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        contentHash: testContent.sample.ipfsSha256,
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_ASSET_INSTANCE_CREATED,
        data,
        blockNumber: '123',
        transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
      }]));
      await eventPromise;
    });

    it('Checks that the asset instance is confirmed', async () => {
      const getAssetInstancesResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}`)
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.ipfsSha256);
      assert.deepStrictEqual(assetInstance.content, testContent.sample.object);
      assert.strictEqual(assetInstance.submitted, undefined);
      assert.strictEqual(assetInstance.receipt, undefined);
      assert.strictEqual(assetInstance.timestamp, timestamp);
      assert.strictEqual(assetInstance.blockNumber, 123);
      assert.strictEqual(assetInstance.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);
    });

  });

});
};
