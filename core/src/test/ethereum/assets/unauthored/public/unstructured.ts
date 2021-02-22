import { app, mockEventStreamWebSocket } from '../../../../common';
import request from 'supertest';
import assert from 'assert';
import nock from 'nock'
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../../../../lib/interfaces';
import * as utils from '../../../../../lib/utils';

export const testUnauthoredPublicUnstructured = () => {

describe('Assets: unauthored - public - unstructured', async () => {

  const assetDefinitionID = 'c4ecf059-67be-4e61-900f-352876604a7f';

  describe('Asset definition', async () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });

      nock('https://ipfs.kaleido.io')
        .get('/ipfs/Qmc6W97aTfV8QuBMkYYygRBakJZ93ZjadtGPNi3KbYALTh')
        .reply(200, {
          assetDefinitionID: assetDefinitionID,
          name: 'unauthored - public - unstructured',
          isContentPrivate: false,
          isContentUnique: true
        });

      const data: IEventAssetDefinitionCreated = {
        author: '0x0000000000000000000000000000000000000002',
        assetDefinitionHash: '0xcc63cbfd00dc7c62c1265a42074afb19531e67b10f85b7f5170b836655a10fd0',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - public - unstructured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.strictEqual(assetDefinition.name, 'unauthored - public - unstructured');
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

});
};
