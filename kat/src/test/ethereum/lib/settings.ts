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
import assert from 'assert';
import request from 'supertest';
import { ISettings } from '../../../lib/interfaces';

export const testSettings = async () => {

describe('Settings', () => {
  it('Checks that settings can be retrieved', async () => {
    const getSettingsResponse = await request(app)
      .get('/api/v1/settings')
      .expect(200);
    const settings: ISettings = getSettingsResponse.body;
    assert.deepStrictEqual(settings.clientEvents, [ 'asset-instance-submitted' ]);
  });

  it('Checks that settings can be updated', async () => {
    const result = await request(app)
      .put('/api/v1/settings')
      .send({
        key: 'clientEvents',
        value: ['asset-instance-property-set']
      })
      .expect(200);
    assert.deepStrictEqual(result.body.status, 'success');

    const getSettingsResponse = await request(app)
      .get('/api/v1/settings')
      .expect(200);
    const settings: ISettings = getSettingsResponse.body;
    assert.deepStrictEqual(settings.clientEvents, [ 'asset-instance-property-set' ]);
  });

  it('Fails when attempting to add an invalid setting', async () => {
    const result = await request(app)
      .put('/api/v1/settings')
      .send({
        key: 'clientEvents',
        value: ['invalid']
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid Settings' });
  });
});
};
