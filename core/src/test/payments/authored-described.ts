import { app, mockEventStreamWebSocket } from '../common';
import { testDescription } from '../samples';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBPaymentDefinition, IEventPaymentDefinitionCreated } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

describe('Payment definitions: authored - described', async () => {

  let paymentDefinitionID: string;

  describe('Create described payment definition', () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the payment definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedPaymentDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=false')
        .reply(200);

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
      assert.strictEqual(paymentDefinition.confirmed, false);
      assert.deepStrictEqual(paymentDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(paymentDefinition.name, 'authored - described');
      assert.strictEqual(typeof paymentDefinition.timestamp, 'number');

      const getPaymentDefinitionResponse = await request(app)
        .get(`/api/v1/payments/definitions/${paymentDefinitionID}`)
        .expect(200);
      assert.deepStrictEqual(paymentDefinition, getPaymentDefinitionResponse.body);
    });

    it('Checks that the event stream notification for confirming the payment definition creation is handled', async () => {
      nock('https://ipfs.kaleido.io')
      .get(`/ipfs/${testDescription.schema.ipfsMultiHash}`)
      .reply(200, testDescription.schema.object);
      
      const eventPromise = new Promise((resolve) => {
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
      assert.strictEqual(paymentDefinition.confirmed, true);
      assert.deepStrictEqual(paymentDefinition.descriptionSchema, testDescription.schema.object);
      assert.strictEqual(paymentDefinition.name, 'authored - described');
      assert.strictEqual(paymentDefinition.timestamp, timestamp);
      assert.strictEqual(paymentDefinition.blockchainData.blockNumber, 123);
      assert.strictEqual(paymentDefinition.blockchainData.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');

      const getPaymentDefinitionResponse = await request(app)
        .get(`/api/v1/payments/definitions/${paymentDefinitionID}`)
        .expect(200);
      assert.deepStrictEqual(paymentDefinition, getPaymentDefinitionResponse.body);
    });

  });

});
