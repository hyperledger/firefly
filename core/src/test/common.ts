import { EventEmitter } from 'events';
import rimraf from 'rimraf';
import nock from 'nock';
import mock from 'mock-require';
import { promises as fs} from 'fs';
import path from 'path';
import assert from 'assert';
import request from 'supertest';
import * as utils from '../lib/utils';
import { IEventMemberRegistered } from "../lib/interfaces";
export let app: Express.Application;
export let mockEventStreamWebSocket: EventEmitter;
export let mockDocExchangeSocketIO = new EventEmitter();

let shutDown: () => void;

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

export const setUp = async (protocol: string) => {
    const sandboxPath = path.join(__dirname, `../../test-resources/sandbox/${protocol}`);
    try {
      await fs.mkdir(sandboxPath, {recursive: true});
    } catch(err) {
      console.error(err);
    }
    await fs.copyFile(path.join(__dirname, '../../test-resources/settings.json'), path.join(sandboxPath, 'settings.json'));
    await fs.copyFile(path.join(__dirname, `../../test-resources/config-${protocol}.json`), path.join(sandboxPath, 'config.json'));
    
    mock('ws', MockWebSocket);
    mock('socket.io-client', {
        connect: () => {
            return mockDocExchangeSocketIO;
        }
    });
    // IPFS
    nock('https://ipfs.kaleido.io')
    .post('/api/v0/version')
    .reply(200, { Version: 1 });

    // Doc exchange REST API
    nock('https://docexchange.kaleido.io')
    .get('/documents')
    .reply(200, { entries: [] });
    const { promise } = require('../app');
  ({ app, shutDown } = await promise);

  const eventPromise = new Promise<void>((resolve) => {
    mockEventStreamWebSocket.once('send', message => {
      assert.strictEqual(message, '{"type":"listen","topic":"dev"}');
      resolve();
    })
  });

  mockEventStreamWebSocket.emit('open');
  mockDocExchangeSocketIO.emit('connect');

  await eventPromise;

  if(protocol === 'corda') {
    await setupSampleMembersCorda();
  } else {
    await setupSampleMembersEthereum();
  }
}

const setupSampleMembersCorda = async () => {
  console.log('Setting up corda members');

    nock('https://apigateway.kaleido.io')
      .post('/registerMember')
      .reply(200);
    await request(app)
      .put('/api/v1/members')
      .send({
        address: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        name: 'Test Member 1',
        app2appDestination: 'kld://app2app_1',
        docExchangeDestination: 'kld://docexchange_1'
      })
    const eventPromise = new Promise<void>((resolve) => {
      mockEventStreamWebSocket.once('send', message => {
        assert.strictEqual(message, '{"type":"ack","topic":"dev"}');
        resolve();
      })
    });
    const dataMember1: any = {
      member: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
      name: 'Test Member 1',
      assetTrailInstanceID: 'service-instance',
      app2appDestination: 'kld://app2app_1',
      docExchangeDestination: 'kld://docexchange_1',
    }
    const dataMember2: any =
    {
      member: 'CN=Node of node2 for env1, O=Kaleido, L=Raleigh, C=US',
      name: 'Test Member 2',
      assetTrailInstanceID: 'service-instance',
      app2appDestination: 'kld://app2app_2',
      docExchangeDestination: 'kld://docexchange_2'
    };
    mockEventStreamWebSocket.emit('message', JSON.stringify([{
      signature: utils.contractEventSignaturesCorda.MEMBER_REGISTERED,
      data: {data: dataMember1},
      stateRef: {
          txhash: "85D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE",
          index: 0
      },
      subId: "sb-f5abe54b-53fb-4f63-8236-f3a8a6bc1c60",
      consumedTime: utils.getTimestamp(),
      recordedTime: null
    }, {
      signature: utils.contractEventSignaturesCorda.MEMBER_REGISTERED,
      data: {data: dataMember2},
      stateRef: {
          txhash: "85D867CC5D19AB40AE46E6262F3C274A6B772D68A0AA522F4C5A96196EAF5FCE",
          index: 0
      },
      subId: "sb-f5abe54b-53fb-4f63-8236-f3a8a6bc1c60",
      consumedTime: utils.getTimestamp(),
      recordedTime: null
    }]));
    await eventPromise;
  };

  const setupSampleMembersEthereum = async () => {
    console.log('Setting up ethereum members');
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
    const eventPromise = new Promise<void>((resolve) => {
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

export const cleanUp = async (protocol: string) => {
    shutDown(); 
    await rimraf(path.join(__dirname, `../../test-resources/sandbox/${protocol}`), {}, ( err ) => {
      if(err) {
        console.error(err);
      }
    });
    mock.stop('ws');
    mock.stop('socket.io-client');
    nock.restore();
};
