// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { app, mockEventStreamWebSocket } from '../../common';
import { v4 as uuidV4 } from 'uuid';
import { testContent } from '../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBAssetDefinition, IDBAssetInstance} from '../../../lib/interfaces';
import * as utils from '../../../lib/utils';

export const testAssetsAuthoredPrivateStructured = () => {

describe('Assets: authored - structured', async () => {

  let assetDefinitionID = uuidV4();
  const assetDefinitionName = 'authored - private - structured';
  const timestamp = new Date();

  describe('Create asset definition', () => {

    it('Checks that the asset definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createAssetDefinition')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: 'Qmf71q7zspRmzvH6yVhkrpWCnK54rvxyj6XSTJ5tgBiZfV' });

      const result = await request(app)
        .post('/api/v1/assets/definitions')
        .send({
          name: assetDefinitionName,
          author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
          assetDefinitionID,
          isContentPrivate: true,
          isContentUnique: true,
          contentSchema: testContent.schema.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');

      const getAssetDefinitionsResponse = await request(app)
        .get('/api/v1/assets/definitions')
        .expect(200);
      const assetDefinition = getAssetDefinitionsResponse.body.find((assetDefinition: IDBAssetDefinition) => assetDefinition.name === 'authored - private - structured');
      assert.strictEqual(assetDefinition.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetDefinition.author, 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US');
      assert.strictEqual(assetDefinition.isContentPrivate, true);
      assert.strictEqual(assetDefinition.isContentUnique, true);
      assert.deepStrictEqual(assetDefinition.contentSchema, testContent.schema.object);
      assert.strictEqual(assetDefinition.name, 'authored - private - structured');
      assert.strictEqual(typeof assetDefinition.submitted, 'number');
    });

  });

  describe('Asset instances', async () => {

    let assetInstanceID: string;

    describe('Asset instances - argument validation', async () => {
      it('Attempting to add an asset instance without specifying participants should raise an error', async () => {
        const result = await request(app)
          .post(`/api/v1/assets/${assetDefinitionID}`)
          .send({
            content: {
              my_content_string: 'test sample content string',
              my_content_number: 124,
              my_content_boolean: false
            },
            author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US'
          })
          .expect(400);
        assert.deepStrictEqual(result.body, { error: 'Missing asset participants' });
      });

      it('Attempting to add an asset instance without specifying participants should raise an error', async () => {
        const result = await request(app)
          .post(`/api/v1/assets/${assetDefinitionID}`)
          .send({
            content: {
              my_content_string: 'test sample content string',
              my_content_number: 124,
              my_content_boolean: false
            },
            author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
            participants: ['CN=Node of node3 for env1, O=Kaleido, L=Raleigh, C=US']
          })
          .expect(400);
        assert.deepStrictEqual(result.body, { error: `One or more participants are not registered` });
      });

      it('Attempting to set an asset instance property without specifying key should raise an error', async () => {
        const result = await request(app)
          .put(`/api/v1/assets/some-asset-def-id/some-asset-id`)
          .send({
            action: 'set-property',
            author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
          })
          .expect(400);
        assert.deepStrictEqual(result.body, { error: `Missing asset property key` });
      });

      it('Attempting to set an asset instance property without specifying value should raise an error', async () => {
        const result = await request(app)
          .put(`/api/v1/assets/some-asset-def-id/some-asset-id`)
          .send({
            action: 'set-property',
            key: 'key',
            author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
          })
          .expect(400);
        assert.deepStrictEqual(result.body, { error: `Missing asset property value` });
      });
    });

    it('Checks that an asset instance can be created', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createAssetInstance')
        .reply(200);

      const result = await request(app)
        .post(`/api/v1/assets/${assetDefinitionID}`)
        .send({
          author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
          content: testContent.sample.object,
          participants: ['CN=Node of node2 for env1, O=Kaleido, L=Raleigh, C=US']
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      assetInstanceID = result.body.assetInstanceID;

      const getAssetInstancesResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}`)
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.docExchangeSha256);
      assert.deepStrictEqual(assetInstance.content, testContent.sample.object);
      assert.strictEqual(typeof assetInstance.submitted, 'number');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);

    });

    it('Checks that the event stream notification for confirming the asset instance creation is handled', async () => {
      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const assetData: any = {
        assetDefinitionID: assetDefinitionID,
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        assetInstanceID: assetInstanceID,
        contentHash: testContent.sample.docExchangeSha256,
        participants: ['CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US', 'CN=Node of node2 for env1, O=Kaleido, L=Raleigh, C=US']
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignaturesCorda.ASSET_INSTANCE_CREATED,
        data: {data: assetData},
        stateRef: {
          txhash: "25D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE",
          index: 0
        },
        subId: "sb-f5abe54b-53fb-4f63-8236-f3a8a6bc1c60",
        recordedTime: timestamp.toISOString(),
        consumedTime: null
      }]));
      await eventPromise;
    });

    it('Checks that the asset instance is confirmed', async () => {
      const getAssetInstancesResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}`)
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.docExchangeSha256);
      assert.deepStrictEqual(assetInstance.content, testContent.sample.object);
      assert.strictEqual(assetInstance.timestamp, timestamp.getTime());
      assert.strictEqual(typeof assetInstance.submitted, 'number');
      assert.strictEqual(assetInstance.transactionHash, '25D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE');
      assert.deepStrictEqual(assetInstance.participants, ['CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US', 'CN=Node of node2 for env1, O=Kaleido, L=Raleigh, C=US'])
      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);
    });

    it('Checks that the asset instance property can be set', async () => {
      nock('https://apigateway.kaleido.io')
      .post('/setAssetInstanceProperty')
      .reply(200);

      const result = await request(app)
        .put(`/api/v1/assets/${assetDefinitionID}/${assetInstanceID}`)
        .send({
          author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
          action: 'set-property',
          key: 'key1',
          value: 'value1'
         })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');

      const getAssetInstancesResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}`)
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US');
      assert.strictEqual(assetInstance.properties[assetInstance.author]['key1'].value, 'value1');
      assert.strictEqual(typeof assetInstance.properties[assetInstance.author]['key1'].submitted, 'number');
      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);

    });

    it('Checks that the event stream notification for confirming the asset instance set property is handled', async () => {
      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const assetData: any = {
        assetDefinitionID: assetDefinitionID,
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        assetInstanceID: assetInstanceID,
        key: 'key1',
        value: 'value1',
        participants: ['CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US', 'CN=Node of node2 for env1, O=Kaleido, L=Raleigh, C=US']
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignaturesCorda.ASSET_PROPERTY_SET,
        data: {data: assetData},
        stateRef: {
          txhash: "35D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE",
          index: 0
        },
        subId: "sb-f5abe54b-53fb-4f63-8236-f3a8a6bc1c60",
        recordedTime: timestamp.toISOString(),
        consumedTime: null
      }]));
      await eventPromise;
    });

    it('Checks that the asset instance property set is confirmed', async () => {
      const getAssetInstancesResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}`)
        .expect(200);
      const assetInstance = getAssetInstancesResponse.body.find((assetInstance: IDBAssetInstance) => assetInstance.assetInstanceID === assetInstanceID);
      assert.strictEqual(assetInstance.author, 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US');
      assert.strictEqual(assetInstance.assetDefinitionID, assetDefinitionID);
      assert.strictEqual(assetInstance.contentHash, testContent.sample.docExchangeSha256);
      assert.deepStrictEqual(assetInstance.content, testContent.sample.object);
      assert.strictEqual(assetInstance.timestamp, timestamp.getTime());
      assert.strictEqual(typeof assetInstance.submitted, 'number');
      assert.strictEqual(assetInstance.transactionHash, '25D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE');
      assert.deepStrictEqual(assetInstance.participants, ['CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US', 'CN=Node of node2 for env1, O=Kaleido, L=Raleigh, C=US'])
      assert.strictEqual(assetInstance.properties[assetInstance.author]['key1'].value, 'value1');
      assert.strictEqual(typeof assetInstance.properties[assetInstance.author]['key1'].submitted, 'number');
      assert.strictEqual(assetInstance.properties[assetInstance.author]['key1'].history[assetInstance.timestamp].transactionHash, '35D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/assets/${assetDefinitionID}/${assetInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(assetInstance, getAssetInstanceResponse.body);
    });

  });

});
};
