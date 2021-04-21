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
