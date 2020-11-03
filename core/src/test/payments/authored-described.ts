import { app, getNextPaymentDefinitionID, mockEventStreamWebSocket, sampleSchemas } from '../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBPaymentDefinition, IEventPaymentDefinitionCreated } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

let paymentDefinitionID = getNextPaymentDefinitionID();

describe('Payment definitions: authored - described', async () => {

  describe('Create described payment definition', () => {

    const timestamp = utils.getTimestamp();

    it('Checks that the payment definition can be added', async () => {

      nock('https://apigateway.kaleido.io')
        .post('/createDescribedPaymentDefinition?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
        .reply(200);

      nock('https://ipfs.kaleido.io')
        .post('/api/v0/add')
        .reply(200, { Hash: sampleSchemas.assetDescription.multiHash });

      const result = await request(app)
        .post('/api/v1/payments/definitions')
        .send({
          name: 'authored - described',
          author: '0x0000000000000000000000000000000000000001',
          amount: 1,
          descriptionSchema: sampleSchemas.assetDescription.object
        })
        .expect(200);
      assert.deepStrictEqual(result.body, { status: 'submitted' });

      const getPaymentDefinitionsResponse = await request(app)
        .get('/api/v1/payments/definitions')
        .expect(200);
      const paymentDefinition = getPaymentDefinitionsResponse.body.find((paymentDefinition: IDBPaymentDefinition) => paymentDefinition.name === 'authored - described');
      assert.strictEqual(paymentDefinition.author, '0x0000000000000000000000000000000000000001');
      assert.strictEqual(paymentDefinition.confirmed, false);
      assert.strictEqual(paymentDefinition.amount, 1);
      assert.deepStrictEqual(paymentDefinition.descriptionSchema, sampleSchemas.assetDescription.object);
      assert.strictEqual(paymentDefinition.name, 'authored - described');
      assert.strictEqual(typeof paymentDefinition.timestamp, 'number');
    });

    it('Checks that the event stream notification for confirming the payment definition creation is handled', async () => {
      const eventPromise = new Promise((resolve) => {
        mockEventStreamWebSocket.once('send', message => {
          assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
          resolve();
        })
      });
      const data: IEventPaymentDefinitionCreated = {
        paymentDefinitionID: paymentDefinitionID.toString(),
        author: '0x0000000000000000000000000000000000000001',
        name: 'authored - described',
        descriptionSchemaHash: sampleSchemas.assetDescription.sha256,
        amount: '1',
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.DESCRIBED_PAYMENT_DEFINITION_CREATED,
        data
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
      assert.strictEqual(paymentDefinition.amount, 1);
      assert.deepStrictEqual(paymentDefinition.descriptionSchema, sampleSchemas.assetDescription.object);
      assert.strictEqual(paymentDefinition.name, 'authored - described');
      assert.strictEqual(paymentDefinition.timestamp, timestamp);

      const getPaymentDefinitionResponse = await request(app)
      .get(`/api/v1/payments/definitions/${paymentDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(paymentDefinition, getPaymentDefinitionResponse.body);
    });

  });


});
