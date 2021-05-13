// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { IDocExchangeTransferData, IDocExchangeListener, IDocExchangeDocumentDetails } from '../lib/interfaces';
import { config } from '../lib/config';
import { Stream, Readable } from 'stream';
import io from 'socket.io-client';
import FormData from 'form-data';
import axios from 'axios';
import * as utils from '../lib/utils';

const log = utils.getLogger('clients/doc-exchange.ts');

let socket: SocketIOClient.Socket
let listeners: IDocExchangeListener[] = [];

export const init = async () => {
  try {
    const response = await axios.get(`${config.docExchange.apiEndpoint}/documents`, {
      auth: {
        username: config.appCredentials.user,
        password: config.appCredentials.password
      }
    });
    if (!Array.isArray(response.data.entries)) {
      throw 'Invalid response';
    } else {
      establishSocketIOConnection();
    }
  } catch (err) {
    throw new Error(`Document exchange REST connection failed. ${err}`);
  }
};

const establishSocketIOConnection = () => {
  let error = false;
  socket = io.connect(config.docExchange.socketIOEndpoint, {
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
      log.info('Document exchange Socket IO connection restored');
    }
  }).on('connect_error', (err: Error) => {
    error = true;
    log.error(`Document exchange Socket IO connection error. ${err.toString()}`);
  }).on('error', (err: Error) => {
    error = true;
    log.error(`Document exchange Socket IO error. ${err.toString()}`);
  }).on('document_received', (transferData: IDocExchangeTransferData) => {
    log.trace(`Doc exchange transfer event ${JSON.stringify(transferData)}`);
    for (const listener of listeners) {
      listener(transferData);
    }
  }) as SocketIOClient.Socket;
};

export const addListener = (listener: IDocExchangeListener) => {
  listeners.push(listener);
};

export const removeListener = (listener: IDocExchangeListener) => {
  listeners = listeners.filter(entry => entry != listener);
};

export const downloadStream = async (documentPath: string): Promise<Buffer> => {
  const response = await axios.get(`${config.docExchange.apiEndpoint}/documents/${documentPath}`, {
    responseType: 'arraybuffer',
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return response.data;
};

export const downloadJSON = async <T>(documentPath: string): Promise<T> => {
  const response = await axios.get(`${config.docExchange.apiEndpoint}/documents/${documentPath}`, {
    responseType: 'json',
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return response.data;
};

export const findDocumentByHash = async (documentHash: string): Promise<string | null> => {
  const result = await axios({
    url: `${config.docExchange.apiEndpoint}/search?query=${documentHash}&by_hash=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  if (result.data.documents.length > 0) {
    return result.data.documents[0].full_path;
  }
  return null;
}

export const uploadString = async (value: string, path: string): Promise<string> => {
  const readable = new Readable();
  readable.push(value);
  readable.push(null);
  return uploadStream(readable, path);
};

export const uploadStream = async (stream: Stream, path: string): Promise<string> => {
  const formData = new FormData();
  formData.append('document', stream);
  const result = await axios({
    method: 'put',
    url: `${config.docExchange.apiEndpoint}/documents/${path}`,
    data: formData,
    headers: formData.getHeaders(),
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return result.data.hash;
};

export const transfer = async (from: string, to: string, document: string) => {
  await axios({
    method: 'post',
    url: `${config.docExchange.apiEndpoint}/transfers`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { from, to, document }
  });
}

export const getDocumentDetails = async (filePath: string): Promise<IDocExchangeDocumentDetails> => {
  const result = await axios({
    url: `${config.docExchange.apiEndpoint}/documents/${filePath}?details_only=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return result.data;
};

export const reset = () => {
  if (socket) {
    log.info('Document exchange Socket IO connection reset');
    socket.close();
    establishSocketIOConnection();
  }
};
