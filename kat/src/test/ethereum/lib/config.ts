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

import * as utils from '../../../lib/utils';
import nock from 'nock';
import sinon from 'sinon';
import assert from 'assert';

export const testUtils = async () => {

describe('Utils', () => {

  before(() => {
    sinon.replace(utils, 'constants', { ...utils.constants, REST_API_CALL_MAX_ATTEMPTS: 3, REST_API_CALL_RETRY_DELAY_MS: 0 });
  });

  describe('Axios with retry', () => {

    it('First attempt successful', async () => {
      nock('https://kaleido.io')
        .get('/test')
        .reply(200, { data: 'test' });
      const result = await utils.axiosWithRetry({
        url: 'https://kaleido.io/test'
      });
      assert.deepStrictEqual(result.data, { data: 'test' });
    });

    it('1 failed attempt', async () => {
      nock('https://kaleido.io')
        .get('/test')
        .reply(500)
        .get('/test')
        .reply(200, { data: 'test' });

      const result = await utils.axiosWithRetry({
        url: 'https://kaleido.io/test'
      });
      assert.deepStrictEqual(result.data, { data: 'test' });
    });

    it('2 failed attempts', async () => {
      nock('https://kaleido.io')
        .get('/test')
        .reply(500)
        .get('/test')
        .reply(500)
        .get('/test')
        .reply(200, { data: 'test' });
      const result = await utils.axiosWithRetry({
        url: 'https://kaleido.io/test'
      });
      assert.deepStrictEqual(result.data, { data: 'test' });
    });

    it('3 failed attempts', async () => {
      nock('https://kaleido.io')
        .get('/test')
        .reply(500)
        .get('/test')
        .reply(500)
        .get('/test')
        .reply(500)
      try {
        await utils.axiosWithRetry({
          url: 'https://kaleido.io/test'
        });
      } catch (err) {
        assert.strictEqual(err.response.status, 500);
      }
    });

    it('Not found should return immediately', async () => {
      nock('https://kaleido.io')
        .get('/test')
        .reply(404);
      try {
        await utils.axiosWithRetry({
          url: 'https://kaleido.io/test'
        });
      } catch (err) {
        assert.deepStrictEqual(err.response.status, 404);
      }
    });

  });

});
};
