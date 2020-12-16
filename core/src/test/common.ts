import { promises as fs } from 'fs';
import path from 'path';
import nock from 'nock';
import mock from 'mock-require';
import { EventEmitter } from 'events';
import assert from 'assert';
import request from 'supertest';
import { IEventMemberRegistered } from '../lib/interfaces';
import * as utils from '../lib/utils';

export let app: Express.Application;
export let mockEventStreamWebSocket: EventEmitter;
export let mockDocExchangeSocketIO = new EventEmitter();

let shutDown: () => void;

before(async () => {

  const sandboxPath = path.join(__dirname, '../../test-resources/sandbox');
  await fs.rmdir(sandboxPath, { recursive: true });
  await fs.mkdir(sandboxPath);
  await fs.copyFile(path.join(__dirname, '../../test-resources/config.json'), path.join(__dirname, '../../test-resources/sandbox/config.json'));

  // IPFS
  nock('https://ipfs.kaleido.io')
    .post('/api/v0/version')
    .reply(200, { Version: 1 });

  // Doc exchange REST API
  nock('https://docexchange.kaleido.io')
    .get('/documents')
    .reply(200, { entries: [] });

  class MockWebSocket extends EventEmitter {

    constructor(url: string) {
      super();
      assert.strictEqual(url, 'ws://eventstreams.kaleido.io');
      mockEventStreamWebSocket = this;
    }

    send(message: string) {
      mockEventStreamWebSocket.emit('send', message);
    }

    ping() { }

    close() { }

  };

  mock('ws', MockWebSocket);

  mock('socket.io-client', {
    connect: (_url: string) => {
      // assert.strictEqual(url, 'http://docexchange.ws.kaleido.io');
      return mockDocExchangeSocketIO;
    }
  });

  const { promise } = require('../app');
  ({ app, shutDown } = await promise);

  const eventPromise = new Promise((resolve) => {
    mockEventStreamWebSocket.once('send', message => {
      assert.strictEqual(message, '{"type":"listen","topic":"dev"}');
      resolve();
    })
  });

  mockEventStreamWebSocket.emit('open');
  mockDocExchangeSocketIO.emit('connect');

  await eventPromise;

  await setupSampleMembers();

});

const setupSampleMembers = async () => {
  nock('https://apigateway.kaleido.io')
    .post('/registerMember?kld-from=0x0000000000000000000000000000000000000001&kld-sync=true')
    .reply(200);
  await request(app)
    .put('/api/v1/members')
    .send({
      address: '0x0000000000000000000000000000000000000001',
      name: 'Test Member 1',
      app2appDestination: 'kld://app2app_1',
      docExchangeDestination: 'kld://docexchange_1'
    })
  const eventPromise = new Promise((resolve) => {
    mockEventStreamWebSocket.once('send', message => {
      assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
      resolve();
    })
  });
  const dataMember1: IEventMemberRegistered = {
    member: '0x0000000000000000000000000000000000000001',
    name: 'Test Member 1',
    assetTrailInstanceID: 'service-instance',
    app2appDestination: 'kld://app2app_1',
    docExchangeDestination: 'kld://docexchange_1',
    timestamp: utils.getTimestamp()
  }
  const dataMember2: IEventMemberRegistered =
  {
    member: '0x0000000000000000000000000000000000000002',
    name: 'Test Member 2',
    assetTrailInstanceID: 'service-instance',
    app2appDestination: 'kld://app2app_2',
    docExchangeDestination: 'kld://docexchange_2',
    timestamp: utils.getTimestamp()
  };
  mockEventStreamWebSocket.emit('message', JSON.stringify([{
    signature: utils.contractEventSignatures.MEMBER_REGISTERED,
    data: dataMember1,
    blockNumber: '123',
    transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
  }, {
    signature: utils.contractEventSignatures.MEMBER_REGISTERED,
    data: dataMember2,
    blockNumber: '123',
    transactionHash: '0x0000000000000000000000000000000000000000000000000000000000000000'
  }]));
  await eventPromise;
};

after(() => {
  shutDown();
});
