import io from 'socket.io-client';
import { createLogger, LogLevelString } from 'bunyan';
import { config } from '../lib/config';
import * as utils from '../lib/utils';
import { AssetTradeMessage, IApp2AppMessage, IApp2AppMessageListener } from '../lib/interfaces';

const log = createLogger({ name: 'clients/app2app.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

let socket: SocketIOClient.Emitter;
let listeners: IApp2AppMessageListener[] = [];

export const init = async () => {
  establishSocketIOConnection();
};

const establishSocketIOConnection = () => {
  let error = false;
  socket = io.connect(config.app2app.socketIOEndpoint, {
    transportOptions: {
      polling: {
        extraHeaders: {
          Authorization: 'Basic ' + Buffer.from(`${config.appCredentials.user}` +
            `:${config.appCredentials.password}`).toString('base64')
        }
      }
    }
  }).on('connect', () => {
    if (error) {
      error = false;
      log.info('App2App messaging Socket IO connection restored');
    }
    socket.emit('subscribe', [config.app2app.destination]);
  }).on('connect_error', (err: Error) => {
    error = true;
    log.error(`App2App messaging Socket IO connection error. ${err.toString()}`);
  }).on('error', (err: Error) => {
    error = true;
    log.error(`App2app messaging Socket IO error. ${err.toString()}`);
  }).on('data', (app2appMessage: IApp2AppMessage) => {
    log.trace(`App2App message ${JSON.stringify(app2appMessage)}`);
    try {
      const content: AssetTradeMessage = JSON.parse(app2appMessage.content);
      for (const listener of listeners) {
        listener(app2appMessage.headers, content);
      }
    } catch (err) {
      log.error(`App2App message error ${err}`);
    }
  });
};

export const addListener = (listener: IApp2AppMessageListener) => {
  listeners.push(listener);
};

export const removeListener = (listener: IApp2AppMessageListener) => {
  listeners = listeners.filter(entry => entry != listener);
};

export const dispatchMessage = (from: string, to: string, content: string) => {
  socket.emit('produce', {
    headers: {
      from,
      to
    },
    content
  });
};