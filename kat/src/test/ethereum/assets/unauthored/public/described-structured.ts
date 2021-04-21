import { app, mockEventStreamWebSocket } from '../../../../common';
import { testDescription, testContent } from '../../../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition, IEventAssetInstanceCreated, IDBAssetInstance } from '../../../../../lib/interfaces';
import * as utils from '../../../../../lib/utils';

export const testUnauthoredPublicDescribedStructured = () => {

describe('Assets: unauthored - public - described - structured', async () => {

  const assetDefinitionID = 'cb289f68-5eaf-46d2-9be1-3dcfdde76885';
  const timestamp = utils.getTimestamp();

  describe('Asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      nock('https://ipfs.kaleido.io')
        .get('/ipfs/Qma49mzceUqCmZrBsFAzC26XnwT1gVzhTBkby2JZsYiaUF')
        .reply(200, {
          assetDefinitionID: assetDefinitionID,
          name: 'unauthored - public - described - structured',
          isContentPrivate: false,
          isContentUnique: true,
          descriptionSchema: testDescription.schema.object,
          contentSchema: testContent.schema.object
        });

      const data: IEventAssetDefinitionCreated = {
        author: '0x0000000000000000000000000000000000000002',
        assetDefinitionHash: '0xae123c4d3b9e77716be55ea8ad616b74ac67c455951698f4e40f22e1ed7981e8',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - public - described - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.name, 'unauthored - public - described - structured');
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

    const assetInstanceID = '9e8acd3b-4067-443f-ad6e-739976bc63ee';

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${testDescription.sample.ipfsMultiHash}`)
      .reply(200, testDescription.sample.object)
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
        descriptionHash: testDescription.sample.ipfsSha256,
        contentHash: testContent.sample.ipfsSha256,
        timestamp: timestamp.toString(),
        isContentPrivate: false
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
      assert.strictEqual(assetInstance.descriptionHash, testDescription.sample.ipfsSha256);
      assert.deepStrictEqual(assetInstance.description, testDescription.sample.object);
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
