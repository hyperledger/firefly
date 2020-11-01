import { promises as fs } from 'fs';
import path from 'path';
import nock from 'nock';
import mock from 'mock-require';
import { EventEmitter } from 'events';
import assert from 'assert';

export let app: Express.Application;
export let shutDown: () => void;
export let mockEventStreamWebSocket: EventEmitter;
export let mockDocExchangeSocketIO = new EventEmitter();

export const setup = async () => {

  const sandboxPath = path.join(__dirname, '../../test/sandbox');
  await fs.rmdir(sandboxPath, { recursive: true });
  await fs.mkdir(sandboxPath);
  await fs.copyFile(path.join(__dirname, '../../test/resources/config.json'), path.join(__dirname, '../../test/sandbox/config.json'));
  process.env.DATA_DIRECTORY = sandboxPath;

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

    close() { }

  };

  mock('ws', MockWebSocket);

  mock('socket.io-client', {
    connect: (url: string) => {
      return mockDocExchangeSocketIO;
    }
  });

  const { promise } = require('../index');
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

}
