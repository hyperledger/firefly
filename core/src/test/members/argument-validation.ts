import { app } from '../common';
import request from 'supertest';
import assert from 'assert';

describe('Members - argument validation', async () => {

  it('Attempting to add a member without an address should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        name: 'Member A',
        app2appDestination: 'kld://app2app',
        docExchangeDestination: 'kld://docexchange'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid member' });
  });

  it('Attempting to add a member without a name should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        address: '0x0000000000000000000000000000000000000001',
        app2appDestination: 'kld://app2app',
        docExchangeDestination: 'kld://docexchange'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid member' });
  });

  it('Attempting to add a member without an app2app destination should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        address: '0x0000000000000000000000000000000000000001',
        name: 'Member A',
        docExchangeDestination: 'kld://docexchange'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid member' });
  });

  it('Attempting to add a member without a document exchange destination should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        address: '0x0000000000000000000000000000000000000001',
        name: 'Member A',
        app2appDestination: 'kld://app2app',
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid member' });
  });

  it('Attempting to get a member that does not exist should raise an error', async () => {
    const result = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000099')
      .expect(404);
    assert.deepStrictEqual(result.body, { error: 'Member not found' });
  });

});
