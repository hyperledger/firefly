import { app, mockEventStreamWebSocket } from '../common';
import request from 'supertest';
import assert from 'assert';
import { IEventPaymentDefinitionCreated, IDBPaymentDefinition, IEventPaymentInstanceCreated, IDBPaymentInstance } from '../../lib/interfaces';
import * as utils from '../../lib/utils';
import nock from 'nock';

describe('Payment definitions: unauthored', async () => {

  const paymentDefinitionID = 'f9812952-50f8-4090-9412-e7b0f3eeb930';
  const timestamp = utils.getTimestamp();

  describe('Payment definition', async () => {

    it('Checks that the event stream notification for confirming the payment definition creation is handled', async () => {

      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventPaymentDefinitionCreated = {
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored',
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.PAYMENT_DEFINITION_CREATED,
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
      const paymentDefinition = getPaymentDefinitionsResponse.body.find((paymentDefinition: IDBPaymentDefinition) => paymentDefinition.name === 'unauthored');
      assert.strictEqual(paymentDefinition.paymentDefinitionID, paymentDefinitionID);
      assert.strictEqual(paymentDefinition.author, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(paymentDefinition.confirmed, true);
      assert.strictEqual(paymentDefinition.name, 'unauthored');
      assert.strictEqual(paymentDefinition.timestamp, timestamp);
      assert.strictEqual(paymentDefinition.blockchainData.blockNumber, 123);
      assert.strictEqual(paymentDefinition.blockchainData.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

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
        .post('/createPaymentInstance?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200);

      const result = await request(app)
        .post('/api/v1/payments/instances')
        .send({
          paymentDefinitionID,
          author: '0x0000000000000000000000000000000000000001',
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
      assert.strictEqual(paymentInstance.recipient, '0x0000000000000000000000000000000000000002');
      assert.strictEqual(paymentInstance.amount, 10);
      assert.strictEqual(paymentInstance.confirmed, false);
      assert.strictEqual(typeof paymentInstance.timestamp, 'number');

      const getPaymentInstanceResponse = await request(app)
        .get(`/api/v1/payments/instances/${paymentInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(paymentInstance, getPaymentInstanceResponse.body);

    });

    it('Checks that the event stream notification for confirming the payment instance creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventPaymentInstanceCreated = {
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        author: '0x0000000000000000000000000000000000000001',
        paymentInstanceID: utils.uuidToHex(paymentInstanceID),
        amount: '10',
        recipient: '0x0000000000000000000000000000000000000002',
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.PAYMENT_INSTANCE_CREATED,
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
      assert.strictEqual(paymentInstance.confirmed, true);
      assert.strictEqual(paymentInstance.amount, 10);
      assert.strictEqual(paymentInstance.timestamp, timestamp);
      assert.strictEqual(paymentInstance.blockchainData.blockNumber, 123);
      assert.strictEqual(paymentInstance.blockchainData.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getAssetInstanceResponse = await request(app)
        .get(`/api/v1/payments/instances/${paymentInstanceID}`)
        .expect(200);
      assert.deepStrictEqual(paymentInstance, getAssetInstanceResponse.body);
    });

  });

});
