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
import { testDescription } from '../../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBPaymentDefinition, IDBPaymentInstance, IEventPaymentDefinitionCreated, IEventPaymentInstanceCreated } from '../../../lib/interfaces';
import * as utils from '../../../lib/utils';

export const testAuthoredDescribed = async () => {

describe('Payment definitions: authored - described', async () => {

  let paymentDefinitionID: string;
  const timestamp = utils.getTimestamp();

  describe('Create described payment definition', () => {

    it('Checks that the payment definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedPaymentDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200, { id: 'my-receipt-id' });

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: testDescription.schema.ipfsMultiHash });

      const result = await request(app)
        .post('/api/v1/payments/definitions')
        .send({
          name: 'authored - described',
          author: '0x0000000000000000000000000000000000000001',
          descriptionSchema: testDescription.schema.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      paymentDefinitionID = result.body.paymentDefinitionID;

      const getPaymentDefinitionsResponse = await request(app)
        .get('/api/v1/payments/definitions')
        .expect(200);
      const paymentDefinition = getPaymentDefinitionsResponse.body.find((paymentDefinition: IDBPaymentDefinition) => paymentDefinition.name === 'authored - described');
      assert.strictEqual(paymentDefinition.paymentDefinitionID, paymentDefinitionID);
      assert.strictEqual(paymentDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.deepStrictEqual(paymentDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(paymentDefinition.name, 'authored - described');
      assert.strictEqual(paymentDefinition.receipt, 'my-receipt-id');
      assert.strictEqual(typeof paymentDefinition.submitted, 'number');

      const getPaymentDefinitionResponse = await request(app)
        .get(`/api/v1/payments/definitions/${paymentDefinitionID}`)
        .expect(200);
      assert.deepStrictEqual(paymentDefinition, getPaymentDefinitionResponse.body);
    });

    it('Checks that the event stream notification for confirming the payment definition creation is handled', async () => {
      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${testDescription.schema.ipfsMultiHash}`)
      .reply(200, testDescription.schema.object);

      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventPaymentDefinitionCreated = {
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - described',
        descriptionSchemaHash: testDescription.schema.ipfsSha256,
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_PAYMENT_DEFINITION_CREATED,
        data,
        blockNumber: '123',
        transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
      }]));
      await eventPromise;
    });

    it('Checks that the payment definition is confirmed', async () => {
      const getPaymentDefinitionsResponse = await request(app)
        .get('/api/v1/payments/definitions')
        .expect(200);
      const paymentDefinition = getPaymentDefinitionsResponse.body.find((paymentDefinition: IDBPaymentDefinition) => paymentDefinition.name === 'authored - described');
      assert.strictEqual(paymentDefinition.paymentDefinitionID, paymentDefinitionID);
      assert.strictEqual(paymentDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.deepStrictEqual(paymentDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(paymentDefinition.name, 'authored - described');
      assert.strictEqual(paymentDefinition.timestamp, timestamp);
      assert.strictEqual(paymentDefinition.receipt, 'my-receipt-id');
      assert.strictEqual(typeof paymentDefinition.submitted, 'number');
      assert.strictEqual(paymentDefinition.blockNumber, 123);
      assert.strictEqual(paymentDefinition.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getPaymentDefinitionResponse = await request(app)
        .get(`/api/v1/payments/definitions/${paymentDefinitionID}`)
        .expect(200);
      assert.deepStrictEqual(paymentDefinition, getPaymentDefinitionResponse.body);
    });

  });

  describe('Payment instances', async () => {

    let paymentInstanceID: string;

    it('Checks that a payment instance can be created', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedPaymentInstance?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200, { id: 'my-receipt-id' });

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: testDescription.sample.ipfsMultiHash })

      const result = await request(app)
        .post('/api/v1/payments/instances')
        .send({
          paymentDefinitionID,
          author: '0x0000000000000000000000000000000000000001',
          description: testDescription.sample.object,
          recipient: '0x0000000000000000000000000000000000000002',
          amount: 10
        })
        .expect(200);
      assert.deepStrictEqual(result.body.status, 'submitted');
      paymentInstanceID = result.body.paymentInstanceID;

      const getPaymentInstancesResponse = await request(app)
        .get('/api/v1/payments/instances')
        .expect(200);
      const paymentInstance = getPaymentInstancesResponse.body.find((paymentInstance: IDBPaymentInstance) => paymentInstance.paymentInstanceID === paymentInstanceID);
      assert.strictEqual(paymentInstance.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(paymentInstance.paymentDefinitionID, paymentDefinitionID);
      assert.strictEqual(paymentInstance.descriptionHash, testDescription.sample.ipfsSha256);
      assert.deepStrictEqual(paymentInstance.description, testDescription.sample.object);
      assert.strictEqual(paymentInstance.recipient, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(paymentInstance.amount, 10);
      assert.strictEqual(paymentInstance.receipt, 'my-receipt-id');
      assert.strictEqual(typeof paymentInstance.submitted, 'number');

      const getPaymentInstanceResponse = await request(app)
        .get(`/api/v1/payments/instances/${paymentInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(paymentInstance, getPaymentInstanceResponse.body);

    });

    it('Checks that the event stream notification for confirming the payment instance creation is handled', async () => {
      const eventPromise = new Promise<void>((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventPaymentInstanceCreated = {
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        author: '0x0000000000000000000000000000000000000001',
        paymentInstanceID: utils.uuidToHex(paymentInstanceID),
        descriptionHash: testDescription.sample.ipfsSha256,
        amount: '10',
        recipient: '0x0000000000000000000000000000000000000002',
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_PAYMENT_INSTANCE_CREATED,
        data,
        blockNumber: '123',
        transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
      }]));
      await eventPromise;
    });

    it('Checks that the payment instance is confirmed', async () => {
      const getAssetInstancesResponse = await request(app)
        .get('/api/v1/payments/instances')
        .expect(200);
      const paymentInstance = getAssetInstancesResponse.body.find((paymentInstance: IDBPaymentInstance) => paymentInstance.paymentInstanceID === paymentInstanceID);
      assert.strictEqual(paymentInstance.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(paymentInstance.recipient, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(paymentInstance.paymentDefinitionID, paymentDefinitionID);
      assert.strictEqual(paymentInstance.descriptionHash, testDescription.sample.ipfsSha256);
      assert.deepStrictEqual(paymentInstance.description, testDescription.sample.object);
      assert.strictEqual(paymentInstance.amount, 10);
      assert.strictEqual(paymentInstance.timestamp, timestamp);
      assert.strictEqual(paymentInstance.receipt, 'my-receipt-id');
      assert.strictEqual(typeof paymentInstance.submitted, 'number');
      assert.strictEqual(paymentInstance.blockNumber, 123);
      assert.strictEqual(paymentInstance.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/payments/instances/${paymentInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(paymentInstance, getAssetInstanceResponse.body);
    });

  });

});
};
