import io from 'socket.io-client';
import { createLogger, LogLevelString } from 'bunyan';
import { config } from '../lib/config';
import * as utils from '../lib/utils';
import { AssetTradeMessage, IApp2AppMessage, IApp2AppMessageListener } from '../lib/interfaces';

const log = createLogger({ name: 'clients/app2app.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

let socket: SocketIOClient.Socket
let internalListeners: IApp2AppMessageListener[] = [];
let externalListeners: IApp2AppMessageListener[] = [];

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
    socket.emit('subscribe', [config.app2app.destinations.internal, config.app2app.destinations.external]);
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
      for (const listener of app2appMessage.headers.to === config.app2app.destinations.internal ? internalListeners : externalListeners) {
        listener(app2appMessage.headers, content);
      }
    } catch (err) {
      log.error(`App2App message error ${err}`);
    }

  }) as SocketIOClient.Socket;
};

export const addListener = (listener: IApp2AppMessageListener, isExternal: boolean) => {
  if (isExternal) {
    externalListeners.push(listener);
  } else {
    internalListeners.push(listener);
  }
};

export const removeListener = (listener: IApp2AppMessageListener, isExternal: boolean) => {
  if (isExternal) {
    externalListeners = externalListeners.filter(entry => entry != listener);
  } else {
    internalListeners = internalListeners.filter(entry => entry != listener);
  }
};

export const dispatchMessage = (to: string, content: string, isExternal: boolean) => {
  socket.emit('produce', {
    headers: {
      from: isExternal ? config.app2app.destinations.external : config.app2app.destinations.internal,
      to
    },
    content
  });
};

export const reset = () => {
  if (socket) {
    log.info('App2App Socket IO connection reset');
    socket.close();
    establishSocketIOConnection();
  }
};