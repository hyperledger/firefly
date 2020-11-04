import { app, getNextPaymentDefinitionID, mockEventStreamWebSocket } from '../common';
import request from 'supertest';
import assert from 'assert';
import { IEventPaymentDefinitionCreated, IDBPaymentDefinition } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

let paymentDefinitionID = getNextPaymentDefinitionID();

describe('Payment definitions: unauthored', async () => {

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
        paymentDefinitionID: paymentDefinitionID.toString(),
        author: '0x0000000000000000000000000000000000000002',
        name: 'unauthored',
        amount: '1',
        timestamp: timestamp.toString()
      };
      mockEventStreamWebSocket.emit('message', JSON.stringify([{
        signature: utils.contractEventSignatures.PAYMENT_DEFINITION_CREATED,
        data
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
      assert.strictEqual(paymentDefinition.amount, 1);
      assert.strictEqual(paymentDefinition.name, 'unauthored');
      assert.strictEqual(paymentDefinition.timestamp, timestamp);

      const getPaymentDefinitionResponse = await request(app)
      .get(`/api/v1/payments/definitions/${paymentDefinitionID}`)
      .expect(200);
      assert.deepStrictEqual(paymentDefinition, getPaymentDefinitionResponse.body);
    });

  });

});
