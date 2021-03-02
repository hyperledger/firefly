import assert from 'assert';
import { createHash, randomBytes } from 'crypto';
import nock from 'nock';
import request from 'supertest';
import { v4 as uuidV4 } from 'uuid';
import { IAssetInstance, IDBAssetDefinition, IDBAssetInstance, IDBBatch, IEventAssetDefinitionCreated, IEventAssetInstanceBatchCreated } from '../../../../../lib/interfaces';
import * as utils from '../../../../../lib/utils';
import { app, mockEventStreamWebSocket } from '../../../../common';
import { testContent, testDescription } from '../../../../samples';

export const testUnauthoredPrivateDescribedStructured = () => {

describe('Assets: unauthored - private - described - structured', async () => {

  const assetDefinitionID = '9ac02d4e-e0a3-4e9e-9d97-be0a41595425';
  const timestamp = utils.getTimestamp();

  let batchMaxRecordsToRestore: number;
  beforeEach(() => {
    nock.cleanAll();
    // Force batches to close immediately
    batchMaxRecordsToRestore = utils.constants.BATCH_MAX_RECORDS;
    utils.constants.BATCH_MAX_RECORDS = 1;
  });

  afterEach(() => {
    assert.deepStrictEqual(nock.pendingMocks(), []);
    utils.constants.BATCH_MAX_RECORDS = batchMaxRecordsToRestore;
  });

  describe('Asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
        .get('/ipfs/QmbGq3hw6r5C3yUkGvqJHq7c67pyMHXxQJ6XBHUh9kf5Ex')
        .reply(200, {
          assetDefinitionID: assetDefinitionID,
          name: 'unauthored - private - described - structured',
          isContentPrivate: true,
          isContentUnique: true,
          descriptionSchema: testDescription.schema.object,
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
        assetDefinitionHash: '0xc02d4bc2c162538e1ca45d20c472b608520d99d6d3988383251d626b8a3411d9',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - private - described - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.deepStrictEqual(assetDefinition.contentSchema,testContent.schema.object);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.name, 'unauthored - private - described - structured');
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

    const assetInstanceID = 'e07f4682-71d1-4a76-8084-8adbc97ef99c';
    const batchHashSha256 = '0x' + createHash('sha256').update(randomBytes(10)).digest().toString('hex');
    const batchHashIPFSMulti = utils.sha256ToIPFSHash(batchHashSha256);

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {

      const testBatch: IDBBatch<IAssetInstance> = {
        author: '0x0000000000000000000000000000000000000002',
        batchID: uuidV4(),
        completed: Date.now() - 100,
        created: Date.now() - 200,
        type: 'asset-instances',
        records: [{
          assetDefinitionID,
          assetInstanceID,
          author: '0x0000000000000000000000000000000000000002',
          description: testDescription.sample.object,
          descriptionHash: '0x' + utils.getSha256(JSON.stringify(testDescription.sample.object)),
          contentHash: '0x' + utils.getSha256(JSON.stringify(testContent.sample.object)),
        }],
      };

      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${batchHashIPFSMulti}`)
      .reply(200, testBatch)

      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetInstanceBatchCreated = {
        batchHash: batchHashSha256,
        author: '0x0000000000000000000000000000000000000002',
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.ASSET_INSTANCE_BATCH_CREATED,
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
      assert.strictEqual(assetInstance.descriptionHash, '0x' + utils.getSha256(JSON.stringify(testDescription.sample.object)));
      assert.deepStrictEqual(assetInstance.description, testDescription.sample.object);
      assert.strictEqual(assetInstance.contentHash, '0x' + utils.getSha256(JSON.stringify(testContent.sample.object)));
      assert.deepStrictEqual(assetInstance.content, undefined);
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
