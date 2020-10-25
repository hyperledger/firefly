import WebSocket from 'ws';
import { config } from '../lib/config';
import { contractEventSignatures } from '../lib/utils';
import { IEventStreamMessage } from '../lib/interfaces';
import * as membersHandler from '../handlers/members';
import { IMemberRegisteredEvent } from '../lib/interfaces';

let ws: WebSocket;

export const init = () => {
  ws = new WebSocket(config.eventStreams.wsEndpoint, {
    headers: {
      Authorization: 'Basic ' + Buffer.from(`${config.appCredentials.user}:${config.appCredentials.password}`).toString('base64')
    }
  }).on('open', () => {
    console.log('Event stream websocket connection open');
    ws.send(JSON.stringify({
      type: 'listen',
      topic: config.eventStreams.topic
    }));
  }).on('close', () => {
    console.log('Event stream websocket connection closed');
  }).on('message', async (message: string) => {
    const messageArray:Array<IEventStreamMessage> = JSON.parse(message);
    for (const message of messageArray) {
      switch(message.signature) {
        case contractEventSignatures.MEMBER_REGISTERED:
          await membersHandler.handleMemberRegisteredEvent(message.data as IMemberRegisteredEvent);
      }
    }
    ws.send(JSON.stringify({
      type: 'ack',
      topic: config.eventStreams.topic
    }))
  })
};
