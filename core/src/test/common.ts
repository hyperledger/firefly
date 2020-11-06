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

export const sampleSchemas = {
  description: {
    object: {
      type: 'object',
      required: ['my_description_string', 'my_description_number', 'my_description_boolean'],
      properties: {
        my_description_string: {
          type: 'string'
        },
        my_description_number: {
          type: 'number'
        },
        my_description_boolean: {
          type: 'boolean'
        }
      }
    },
    sha256: '0xbde03e0e77b5422ff3ce4889752ac9450343420a6a4354542b9fd14fd5fa435c',
    multiHash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io',
    sample: {
      my_description_string: 'sample description string',
      my_description_number: 123,
      my_description_boolean: true
    }
  },
  content: {
    object: {
      type: 'object',
      required: ['my_content_string', 'my_content_number', 'my_content_boolean'],
      properties: {
        my_content_string: {
          type: 'string'
        },
        my_content_number: {
          type: 'number'
        },
        my_content_boolean: {
          type: 'boolean'
        }
      }
    },
    sha256: '0x64c97929fb90da1b94d560a29d8522c77b6c662588abb6ad23f1a0377250a2b0',
    multiHash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw',
    sample: {
      my_content_string: 'sample content string',
      my_content_number: 456,
      my_content_boolean: true
    }
  }
}

let assetDefinitionID = 0;
export const getNextAssetDefinitionID = () => {
  return assetDefinitionID++;
}

let paymentDefinitionID = 0;
export const getNextPaymentDefinitionID = () => {
  return paymentDefinitionID++;
}

let shutDown: () => void;

before(async () => {

  const sandboxPath = path.join(__dirname, '../../test-resources/sandbox');
  await fs.rmdir(sandboxPath, { recursive: true });
  await fs.mkdir(sandboxPath);
  await fs.copyFile(path.join(__dirname, '../../test-resources/config.json'), path.join(__dirname, '../../test-resources/sandbox/config.json'));

  // API Gateway
  nock('https://apigateway.kaleido.io')
    .get('/getStatus')
    .reply(200,
      {
        totalAssetDefinitions: '0',
        totalPaymentDefinitions: '0'
      });

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
    connect: (_url: string) => {
      // assert.strictEqual(url, 'http://docexchange.ws.kaleido.io');
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
    app2appDestination: 'kld://app2app_1',
    docExchangeDestination: 'kld://docexchange_1',
    timestamp: utils.getTimestamp()
  }
  const dataMember2: IEventMemberRegistered =
  {
    member: '0x0000000000000000000000000000000000000002',
    name: 'Test Member 2',
    app2appDestination: 'kld://app2app_2',
    docExchangeDestination: 'kld://docexchange_2',
    timestamp: utils.getTimestamp()
  };
  mockEventStreamWebSocket.emit('message', JSON.stringify([{
    signature: utils.contractEventSignatures.MEMBER_REGISTERED,
    data: dataMember1
  }, {
    signature: utils.contractEventSignatures.MEMBER_REGISTERED,
    data: dataMember2
  }]));
  await eventPromise;
};

after(() => {
  shutDown();
});
