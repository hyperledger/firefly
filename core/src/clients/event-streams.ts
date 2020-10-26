import WebSocket from 'ws';
import { config } from '../lib/config';
import * as utils from '../lib/utils';
import { IEventStreamMessage } from '../lib/interfaces';
import * as membersHandler from '../handlers/members';
import { IMemberRegisteredEvent } from '../lib/interfaces';

let ws: WebSocket;
let disconnectionDetected = false;

export const init = () => {
  ws = new WebSocket(config.eventStreams.wsEndpoint, {
    headers: {
      Authorization: 'Basic ' + Buffer.from(`${config.appCredentials.user}:${config.appCredentials.password}`).toString('base64')
    }
  });
  addEventHandlers();
};

const addEventHandlers = () => {
  ws.on('open', () => {
    if(disconnectionDetected) {
      disconnectionDetected = false;
      console.log('Event stream websocket connected');
    }
    ws.send(JSON.stringify({
      type: 'listen',
      topic: config.eventStreams.topic
    }));
  }).on('close', () => {
    disconnectionDetected = true;
    console.log(`Event stream websocket disconnected, attempting to reconnect in ${utils.constants.EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS} second(s)`);
    setTimeout(() => {
      init();
    }, utils.constants.EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS * 1000);
  }).on('message', async (message: string) => {
    const messageArray:Array<IEventStreamMessage> = JSON.parse(message);
    for (const message of messageArray) {
      switch(message.signature) {
        case utils.contractEventSignatures.MEMBER_REGISTERED:
          await membersHandler.handleMemberRegisteredEvent(message.data as IMemberRegisteredEvent);
      }
    }
    ws.send(JSON.stringify({
      type: 'ack',
      topic: config.eventStreams.topic
    }))
  }).on('error', err => {
    console.log(`Event stream websocket error. ${err}`);
  });
};