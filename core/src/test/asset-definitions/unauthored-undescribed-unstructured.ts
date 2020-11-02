import { app, mockEventStreamWebSocket } from '../common';
import request from 'supertest';
import assert from 'assert';
import { IEventAssetDefinitionCreated, IDBAssetDefinition } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Asset definitions: unauthored - undescribed - unstructured', async () => {

  describe('Public asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: '8',
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - undescribed - unstructured - public',
        isContentPrivate: false,
        timestamp: '9'
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - undescribed - unstructured - public');
      assert.strictEqual(assetDefinition.assetDefinitionID, 8);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, false);
      assert.strictEqual(assetDefinition.name, 'unauthored - undescribed - unstructured - public');
      assert.strictEqual(assetDefinition.timestamp, 9);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/8')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

  describe('Private asset definition', async () => {

    it('Checks that the event stream notification for confirming the asset definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventAssetDefinitionCreated = {
        assetDefinitionID: '9',
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored - undescribed - unstructured - private',
        isContentPrivate: true,
        timestamp: '10'
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
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'unauthored - undescribed - unstructured - private');
      assert.strictEqual(assetDefinition.assetDefinitionID, 9);
      assert.strictEqual(assetDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(assetDefinition.confirmed, true);
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.name, 'unauthored - undescribed - unstructured - private');
      assert.strictEqual(assetDefinition.timestamp, 10);

      const getAssetDefinitionResponse = await request(app)
      .get('/api/v1/assets/definitions/9')
      .expect(200);
      assert.deepStrictEqual(assetDefinition, getAssetDefinitionResponse.body);
    });

  });

});
