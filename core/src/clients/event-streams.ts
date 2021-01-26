import WebSocket from 'ws';
import { config } from '../lib/config';
import * as utils from '../lib/utils';
import { IDBBlockchainData, IEventAssetDefinitionCreated, IEventAssetInstanceBatchCreated, IEventAssetInstanceCreated, IEventAssetInstancePropertySet, IEventPaymentDefinitionCreated, IEventPaymentInstanceCreated, IEventStreamMessage } from '../lib/interfaces';
import * as membersHandler from '../handlers/members';
import * as assetDefinitionsHandler from '../handlers/asset-definitions';
import * as paymentDefinitionsHandler from '../handlers/payment-definitions';
import * as assetInstancesHandler from '../handlers/asset-instances';
import * as paymentInstanceHandler from '../handlers/payment-instances';
import { IEventMemberRegistered } from '../lib/interfaces';
import { createLogger, LogLevelString } from 'bunyan';

const log = createLogger({ name: 'clients/event-streams.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

let ws: WebSocket;
let heartBeatTimeout: NodeJS.Timeout;
let disconnectionDetected = false;
let disconnectionTimeout: NodeJS.Timeout;

export const init = () => {
  ws = new WebSocket(config.eventStreams.wsEndpoint, {
    headers: {
      Authorization: 'Basic ' + Buffer.from(`${config.appCredentials.user}:${config.appCredentials.password}`).toString('base64')
    }
  });
  addEventHandlers();
};

export const shutDown = () => {
  if (disconnectionTimeout) {
    clearTimeout(disconnectionTimeout);
  }
  if (ws) {
    clearTimeout(heartBeatTimeout);
    ws.close();
  }
};

const addEventHandlers = () => {
  ws.on('open', () => {
    if (disconnectionDetected) {
      disconnectionDetected = false;
      log.info('Event stream websocket restored');
    }
    ws.send(JSON.stringify({
      type: 'listen',
      topic: config.eventStreams.topic
    }));
    heartBeat();
  }).on('close', () => {
    disconnectionDetected = true;
    log.error(`Event stream websocket disconnected, attempting to reconnect in ${utils.constants.EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS} second(s)`);
    disconnectionTimeout = setTimeout(() => {
      init();
    }, utils.constants.EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS * 1000);
  }).on('message', async (message: string) => {
    await handleMessage(message);
    ws.send(JSON.stringify({
      type: 'ack',
      topic: config.eventStreams.topic
    }));
  }).on('pong', () => {
    heartBeat();
  }).on('error', err => {
    log.error(`Event stream websocket error. ${err}`);
  });
};

const heartBeat = () => {
  ws.ping();
  clearTimeout(heartBeatTimeout);
  heartBeatTimeout = setTimeout(() => {
    log.error('Event stream ping timeout');
    ws.terminate();
  }, utils.constants.EVENT_STREAM_PING_TIMEOUT_SECONDS * 1000);
}

const handleMessage = async (message: string) => {
  const messageArray: Array<IEventStreamMessage> = JSON.parse(message);
  log.info(`Event batch (${messageArray.length})`)
  for (const message of messageArray) {
    log.trace(`Event ${JSON.stringify(message)}`);
    const blockchainData: IDBBlockchainData = {
      blockNumber: Number(message.blockNumber),
      transactionHash: message.transactionHash
    }
    try {
      switch (message.signature) {
        case utils.contractEventSignatures.MEMBER_REGISTERED:
          await membersHandler.handleMemberRegisteredEvent(message.data as IEventMemberRegistered, blockchainData); break;
        case utils.contractEventSignatures.DESCRIBED_STRUCTURED_ASSET_DEFINITION_CREATED:
        case utils.contractEventSignatures.DESCRIBED_UNSTRUCTURED_ASSET_DEFINITION_CREATED:
        case utils.contractEventSignatures.STRUCTURED_ASSET_DEFINITION_CREATED:
        case utils.contractEventSignatures.UNSTRUCTURED_ASSET_DEFINITION_CREATED:
          await assetDefinitionsHandler.handleAssetDefinitionCreatedEvent(message.data as IEventAssetDefinitionCreated, blockchainData); break;
        case utils.contractEventSignatures.DESCRIBED_PAYMENT_DEFINITION_CREATED:
        case utils.contractEventSignatures.PAYMENT_DEFINITION_CREATED:
          await paymentDefinitionsHandler.handlePaymentDefinitionCreatedEvent(message.data as IEventPaymentDefinitionCreated, blockchainData); break;
        case utils.contractEventSignatures.ASSET_INSTANCE_CREATED:
        case utils.contractEventSignatures.DESCRIBED_ASSET_INSTANCE_CREATED:
          await assetInstancesHandler.handleAssetInstanceCreatedEvent(message.data as IEventAssetInstanceCreated, blockchainData); break;
        case utils.contractEventSignatures.ASSET_INSTANCE_BATCH_CREATED:
          await assetInstancesHandler.handleAssetInstanceBatchCreatedEvent(message.data as IEventAssetInstanceBatchCreated, blockchainData); break;
        case utils.contractEventSignatures.DESCRIBED_PAYMENT_INSTANCE_CREATED:
        case utils.contractEventSignatures.PAYMENT_INSTANCE_CREATED:
          await paymentInstanceHandler.handlePaymentInstanceCreatedEvent(message.data as IEventPaymentInstanceCreated, blockchainData); break;
        case utils.contractEventSignatures.ASSET_PROPERTY_SET:
          await assetInstancesHandler.handleSetAssetInstancePropertyEvent(message.data as IEventAssetInstancePropertySet, blockchainData); break;
      }
    } catch (err) {
      log.error(`Failed to handle event: ${message.signature} for message: ${JSON.stringify(message)} with error`, err.stack);
    }
  }
};