import { app } from '../../common';
import request from 'supertest';
import assert from 'assert';

export const testMembersArgumentValidation = async () => {

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
    assert.deepStrictEqual(result.body, { error: 'Missing member address' });
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
    assert.deepStrictEqual(result.body, { error: 'Missing member name' });
  });

  it('Attempting to get a member that does not exist should raise an error', async () => {
    const result = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000099')
      .expect(404);
    assert.deepStrictEqual(result.body, { error: 'Member not found' });
  });

});
};
