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
