import { promises as fs } from 'fs';
import path from 'path';
import nock from 'nock';
import mock from 'mock-require';
import { EventEmitter } from 'events';
import assert from 'assert';

export let app: Express.Application;
export let mockEventStreamWebSocket: EventEmitter;
export let mockDocExchangeSocketIO = new EventEmitter();
export const sampleSchemas = {
  assetDescription: {
    object: {
      type: 'object',
      required: ['my_description_string, my_description_number, my_description_boolean'],
      properties: {
        my_string: {
          type: 'string'
        },
        my_number: {
          type: 'number'
        },
        my_boolean: {
          type: 'boolean'
        }
      }
    },
    sha256: '0xbde03e0e77b5422ff3ce4889752ac9450343420a6a4354542b9fd14fd5fa435c',
    multiHash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io'
  },
  assetContent: {
    object: {
      type: 'object',
      required: ['my_content_string, my_content_number, my_content_boolean'],
      properties: {
        my_string: {
          type: 'string'
        },
        my_number: {
          type: 'number'
        },
        my_boolean: {
          type: 'boolean'
        }
      }
    },
    sha256: '0x64c97929fb90da1b94d560a29d8522c77b6c662588abb6ad23f1a0377250a2b0',
    multiHash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw'
  }
}

let shutDown: () => void;

before(async () => {

  const sandboxPath = path.join(__dirname, '../../test/sandbox');
  await fs.rmdir(sandboxPath, { recursive: true });
  await fs.mkdir(sandboxPath);
  await fs.copyFile(path.join(__dirname, '../../test/resources/config.json'), path.join(__dirname, '../../test/sandbox/config.json'));

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

});

after(() => {
  shutDown();
});