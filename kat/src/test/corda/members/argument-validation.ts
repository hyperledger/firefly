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

  it('Attempting to add a member without a assetTrailInstanceID should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        name: 'Member A',
        address: '0x0000000000000000000000000000000000000001',
        app2appDestination: 'kld://app2app',
        docExchangeDestination: 'kld://docexchange'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing member assetTrailInstanceID' });
  });

  it('Attempting to add a member without a docExchangeDestination should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        name: 'Member A',
        address: '0x0000000000000000000000000000000000000001',
        app2appDestination: 'kld://app2app',
        assetTrailInstanceID: 'asset-instance-a'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing member docExchangeDestination' });
  });

  it('Attempting to add a member without a app2appDestination should raise an error', async () => {
    const result = await request(app)
      .put('/api/v1/members')
      .send({
        name: 'Member A',
        address: '0x0000000000000000000000000000000000000001',
        assetTrailInstanceID: 'asset-instance-a',
        docExchangeDestination: 'kld://docexchange'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing member app2appDestination' });
  });

  it('Attempting to get a member that does not exist should raise an error', async () => {
    const result = await request(app)
      .get('/api/v1/members/0x0000000000000000000000000000000000000099')
      .expect(404);
    assert.deepStrictEqual(result.body, { error: 'Member not found' });
  });

});
};
