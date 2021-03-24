import io from 'socket.io-client';
import { config } from '../lib/config';
import * as utils from '../lib/utils';
import { AssetTradeMessage, IApp2AppMessage, IApp2AppMessageListener } from '../lib/interfaces';

const log = utils.getLogger('clients/app2app.ts');

let socket: SocketIOClient.Socket
let listeners: IApp2AppMessageListener[] = [];

export const init = async () => {
  socket = _establishSocketIOConnection();
};

function subscribeWithRetry() {
  log.trace(`App2App subscription: ${config.app2app.destinations.kat}`)
  socket.emit('subscribe', [config.app2app.destinations.kat], (err: any, data: any) => {
    if (err) {
      log.error(`App2App subscription failure (retrying): ${err}`);
      setTimeout(subscribeWithRetry, utils.constants.SUBSCRIBE_RETRY_INTERVAL);
      return;
    }
    log.trace(`App2App subscription succeeded: ${JSON.stringify(data)}`);
  });
}

export const _establishSocketIOConnection = () => {
  let error = false;
  const s = io.connect(`${config.app2app.socketIOEndpoint}?auto_commit=false&read_ahead=50`, {
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
    subscribeWithRetry();
  }).on('connect_error', (err: Error) => {
    error = true;
    log.error(`App2App messaging Socket IO connection error. ${err.toString()}`);
  }).on('error', (err: Error) => {
    error = true;
    log.error(`App2app messaging Socket IO error. ${err.toString()}`);
  }).on('exception', (err: Error, extra?: any) => {
    // Exceptions are such things as delivery failures. They do not put the connection in error state
    log.error(`App2app messaging exception. ${err.toString()}`, extra);
  }).on('data', (app2appMessage: IApp2AppMessage) => {
    log.trace(`App2App message ${JSON.stringify(app2appMessage)}`);
    try {
      const content: AssetTradeMessage = JSON.parse(app2appMessage.content);
      log.trace(`App2App message type=${content.type}`)
      for (const listener of listeners) {
        listener(app2appMessage.headers, content);
      }
    } catch (err) {
      log.error(`App2App message error ${err}`);
    } finally {
      s.emit('commit');
    }
  }) as SocketIOClient.Socket;
  return s;
};

export const addListener = (listener: IApp2AppMessageListener) => {
  listeners.push(listener);
};

export const removeListener = (listener: IApp2AppMessageListener) => {
  listeners = listeners.filter(entry => entry != listener);
};

export const dispatchMessage = (to: string, content: any) => {
  log.trace(`App2App dispatch type=${content.type}`)
  socket.emit('produce', {
    headers: {
      from: config.app2app.destinations.kat,
      to
    },
    content: JSON.stringify(content)
  }, 'kat', (err: any) => {
    if(err) {
      log.error(`Failed to dispatch App2App message.`, err);
    }
  });
};

export const reset = () => {
  if (socket) {
    log.info('App2App Socket IO connection reset');
    socket.close();
    socket = _establishSocketIOConnection();
  }
};