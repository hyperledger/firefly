import { app, mockEventStreamWebSocket } from '../../common';
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IDBMember } from '../../../lib/interfaces';
import * as utils from '../../../lib/utils';

export const testMemberRegistration = async () => {

describe('Members - registration', async () => {

  const timestampCreation = new Date();
  const timestampUpdate = new Date();

  it('Checks that adding a member sends a request to API Gateway and updates the database', async () => {

    nock('https://apigateway.kaleido.io')
      .post('/registerMember')
      .reply(200);
    const addMemberResponse = await request(app)
      .put('/api/v1/members')
      .send({
        address: 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US',
        name: 'Member 1'
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

  it('Checks that event stream notification for confirming member registrations is handled', async () => {

    const eventPromise = new Promise<void>((resolve) => {
      mockEventStreamWebSocket.once('send', message => {
        assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
        resolve();
      })
    });

    const dataMember: any = {
      member: 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US',
      name: 'Member 1',
      assetTrailInstanceID: 'service-id',
      app2appDestination: 'kld://app2app/internal',
      docExchangeDestination: 'kld://docstore/dest',
    };
    mockEventStreamWebSocket.emit('message', JSON.stringify([{
      signature: utils.contractEventSignaturesCorda.MEMBER_REGISTERED,
      data: {data: dataMember},
      stateRef: {
          txhash: "85D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE",
          index: 0
      },
      subId: "sb-f5abe54b-53fb-4f63-8236-f3a8a6bc1c60",
      consumedTime: null,
      recordedTime: timestampCreation.toISOString()
    }]));
    await eventPromise;
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
    assert.strictEqual(member.transactionHash, '85D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE');
    assert.strictEqual(member.timestamp, timestampCreation.getTime());

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

  it('Checks that updating a member sends a request to API Gateway and updates the database', async () => {
    nock('https://apigateway.kaleido.io')
      .post('/registerMember')
      .reply(200);
    const addMemberResponse = await request(app)
      .put('/api/v1/members')
      .send({
        address: 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US',
        name: 'Member 2'
      })
      .expect(200);
    assert.deepStrictEqual(addMemberResponse.body, { status: 'submitted' });

    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.address, 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.name, 'Member 2');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(member.transactionHash, '85D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE');
    assert.strictEqual(typeof member.timestamp, 'number');

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

  it('Checks that event stream notification for confirming member registrations are handled', async () => {

    const eventPromise = new Promise<void>((resolve) => {
      mockEventStreamWebSocket.once('send', message => {
        assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
        resolve();
      })
    });

    const dataMember: any = {
      member: 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US',
      name: 'Member 2',
      assetTrailInstanceID: 'service-id',
      app2appDestination: 'kld://app2app/internal',
      docExchangeDestination: 'kld://docstore/dest',
    };
    mockEventStreamWebSocket.emit('message', JSON.stringify([{
      signature: utils.contractEventSignaturesCorda.MEMBER_REGISTERED,
      data: {data: dataMember},
      stateRef: {
          txhash: "95D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE",
          index: 0
      },
      subId: "sb-f5abe54b-53fb-4f63-8236-f3a8a6bc1c60",
      consumedTime: null,
      recordedTime: timestampUpdate.toISOString()
    }]));
    await eventPromise;
  });

  it('Get member should return the confirmed member', async () => {
    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.address, 'CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US');
    assert.strictEqual(member.name, 'Member 2');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(member.transactionHash, '95D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE');
    assert.strictEqual(member.timestamp, timestampUpdate.getTime());

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/CN=Node of member1 for env1, O=Kaleido, L=Raleigh, C=US')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

});
};
