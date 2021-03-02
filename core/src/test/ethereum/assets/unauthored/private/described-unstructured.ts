<<<<<<< HEAD:core/src/test/ethereum/assets/unauthored/private/described-unstructured.ts
import { app, mockEventStreamWebSocket } from '../../../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../../../../lib/interfaces';
import * as utils from '../../../../../lib/utils';
import { testDescription } from '../../../../samples';

export const testUnauthoredPrivateDescribedUnstructured = () => {
=======
import { app, mockEventStreamWebSocket } from '../../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../../../lib/interfaces';
import * as utils from '../../../../lib/utils';
import { testDescription } from '../../../samples';
>>>>>>> master:core/src/test/assets/unauthored/private/described-unstructured.ts

describe('Assets: unauthored - private - described - unstructured', async () => {

  const assetDefinitionID = '0989dfe5-4b4c-43e0-91fc-3fea194a7c56';

  describe('Asset definition', async () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {

      nock('https://ipfs.kaleido.io')
        .get('/ipfs/QmbUq3WLWyrXcEtBH8LTpiq3Usg5mkWb4kAGPigg4a7YKU')
        .reply(200, {
          assetDefinitionID: assetDefinitionID,
          name: 'unauthored - private - described - unstructured',
          isContentPrivate: true,
          isContentUnique: true,
          descriptionSchema: testDescription.schema.object,
        });

      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        author: '0x0000000000000000000000000000000000000002',
        assetDefinitionHash: '0xc34043022814b07d606980cbd6d605952943faddadac9f32419ee33c2bdf0eab',
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - private - described - unstructured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.deepStrictEqual(assetDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.name, 'unauthored - private - described - unstructured');
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
