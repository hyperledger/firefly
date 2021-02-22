import { app } from '../../common';
import request from 'supertest';
import assert from 'assert';

export const testPaymentArgumentValidation = async () => {

describe('Payment definitions - argument validation', async () => {

  it('Attempting to get a payment definition that does not exist should raise an error', async () => {
    const result = await request(app)
      .get('/api/v1/payments/definitions/missing')
      .expect(404);
    assert.deepStrictEqual(result.body, { error: 'Payment definition not found' });
  });

  it('Attempting to add a payment definition without a name should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/payments/definitions')
      .send({
        author: '0x0000000000000000000000000000000000000001'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid payment definition name' });
  });

  it('Attempting to add a payment definition without an author should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/payments/definitions')
      .send({
        name: 'My payment definition'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid payment definition author' });
  });

  it('Attempting to add a payment definition with an invalid description schema should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/payments/definitions')
      .send({
        name: 'My payment definition',
        author: '0x0000000000000000000000000000000000000001',
        descriptionSchema: 'INVALID'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid description schema' });
  });

});
};
