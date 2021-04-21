import { app } from '../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBMember } from '../../../lib/interfaces';

export const testMemberRegistration = async () => {

describe('Members - registration', async () => {
  it('Checks that adding a member sends a request to API Gateway and updates the database', async () => {

    nock('https://apigateway.kaleido.io')
      .post('/registerMember')
      .reply(200);
    const addMemberResponse = await request(app)
      .put('/api/v1/members')
      .send({
        address: 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US',
        name: 'Member 1',
        assetTrailInstanceID: 'service-id',
        app2appDestination: 'kld://app2app/internal',
        docExchangeDestination:'kld://docstore/dest'
      })
      .expect(200);
    assert.deepStrictEqual(addMemberResponse.body, { status: 'submitted' });

    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.address, 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.name, 'Member 1');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(typeof member.submitted, 'number');

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

  it('Get member should return the confirmed member', async () => {
    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.address, 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.name, 'Member 1');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

});
};
