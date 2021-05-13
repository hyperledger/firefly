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
import nock from 'nock';
import request from 'supertest';
import assert from 'assert';
import { IEventMemberRegistered, IDBMember } from '../../../lib/interfaces';
import * as utils from '../../../lib/utils';

export const testMemberRegistration = async () => {

describe('Members - registration', async () => {

  const timestampCreation = utils.getTimestamp();
  const timestampUpdate = utils.getTimestamp();

  it('Checks that adding a member sends a request to API Gateway and updates the database', async () => {

    nock('https://apigateway.kaleido.io')
      .post('/registerMember?kld-from=0x0000000000000000000000000000000000000011&kld-sync=false')
      .reply(200);
    const addMemberResponse = await request(app)
      .put('/api/v1/members')
      .send({
        address: '0x0000000000000000000000000000000000000011',
        name: 'Member 1'
      })
      .expect(200);
    assert.deepStrictEqual(addMemberResponse.body, { status: 'submitted' });

    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.address, '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.name, 'Member 1');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(typeof member.submitted, 'number');

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000011')
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

    const data: IEventMemberRegistered = {
      member: '0x0000000000000000000000000000000000000011',
      name: 'Member 1',
      assetTrailInstanceID: 'service-id',
      app2appDestination: 'kld://app2app/internal',
      docExchangeDestination: 'kld://docstore/dest',
      timestamp: timestampCreation
    };
    mockEventStreamWebSocket.emit('message', JSON.stringify([{
      signature: utils.contractEventSignatures.MEMBER_REGISTERED,
      data,
      blockNumber: '123',
      transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
    }]));
    await eventPromise;
  });

  it('Get member should return the confirmed member', async () => {
    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.address, '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.name, 'Member 1');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(member.blockNumber, 123);
    assert.strictEqual(member.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');
    assert.strictEqual(member.timestamp, timestampCreation);

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000011')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

  it('Checks that updating a member sends a request to API Gateway and updates the database', async () => {
    nock('https://apigateway.kaleido.io')
      .post('/registerMember?kld-from=0x0000000000000000000000000000000000000011&kld-sync=false')
      .reply(200);
    const addMemberResponse = await request(app)
      .put('/api/v1/members')
      .send({
        address: '0x0000000000000000000000000000000000000011',
        name: 'Member 2'
      })
      .expect(200);
    assert.deepStrictEqual(addMemberResponse.body, { status: 'submitted' });

    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.address, '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.name, 'Member 2');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(member.blockNumber, 123);
    assert.strictEqual(member.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000000');
    assert.strictEqual(typeof member.timestamp, 'number');

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000011')
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

    const data: IEventMemberRegistered = {
      member: '0x0000000000000000000000000000000000000011',
      name: 'Member 2',
      assetTrailInstanceID: 'service-id',
      app2appDestination: 'kld://app2app/internal',
      docExchangeDestination: 'kld://docstore/dest',
      timestamp: timestampUpdate
    };
    mockEventStreamWebSocket.emit('message', JSON.stringify([{
      signature: utils.contractEventSignatures.MEMBER_REGISTERED,
      data,
      blockNumber: '456',
      transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000001'
    }]));
    await eventPromise;
  });

  it('Get member should return the confirmed member', async () => {
    const getMemberResponse = await request(app)
      .get('/api/v1/members')
      .expect(200);
    const member = getMemberResponse.body.find((member: IDBMember) => member.address === '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.address, '0x0000000000000000000000000000000000000011');
    assert.strictEqual(member.name, 'Member 2');
    assert.strictEqual(member.assetTrailInstanceID, 'service-id');
    assert.strictEqual(member.app2appDestination, 'kld://app2app/internal');
    assert.strictEqual(member.docExchangeDestination, 'kld://docstore/dest');
    assert.strictEqual(member.blockNumber, 456);
    assert.strictEqual(member.transactionHash, '0x0000000000000000000000000000000000000000000000000000000000000001');
    assert.strictEqual(member.timestamp, timestampUpdate);

    const getMemberByAddressResponse = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000011')
      .expect(200);
    assert.deepStrictEqual(member, getMemberByAddressResponse.body);
  });

});
};
