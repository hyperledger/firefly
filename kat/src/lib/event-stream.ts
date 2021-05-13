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

import * as QueryString from 'querystring';
import { URL } from 'url';
import axios, { AxiosInstance } from 'axios';
import { promisify } from 'util';
import * as timers from 'timers';
import * as database from '../clients/database';
const sleep = promisify(timers.setTimeout);

import * as utils from './utils';
import { config } from './config'
import { IConfig, IEventStream, IEventStreamSubscription } from './interfaces';
const logger = utils.getLogger('lib/event-stream.ts');

/* istanbul ignore next */
const requestLogger = (config: any) => {
  const qs = config.params ? `?${QueryString.stringify(config.params)}` : '';
  logger.info(`--> ${config.method} ${config.baseURL}${config.url}${qs}`);
  logger.debug(config.data);
  return config;
};

/* istanbul ignore next */
const responseLogger = (response: any) => {
  const { config, status, data } = response;
  logger.info(`<-- ${config.method} ${config.url} [${status}]`);
  logger.debug(data);
  return response;
};

/* istanbul ignore next */
const errorLogger = (err: any) => {
  const { config = {}, response = {} } = err;
  const { status, data } = response;
  logger.info(`<-- ${config.method} ${config.url} [${status || err}]: ${JSON.stringify(data)}`);
  throw err;
};

const subscriptionsInfoEthereum = [
  ['Asset instance created', 'AssetInstanceCreated'],
  ['Asset instance batch created', 'AssetInstanceBatchCreated'],
  ['Payment instance created', 'PaymentInstanceCreated'],
  ['Payment definition created', 'PaymentDefinitionCreated'],
  ['Asset definition created', 'AssetDefinitionCreated'],
  ['Asset instance property set', 'AssetInstancePropertySet'],
  ['Described payment instance created', 'DescribedPaymentInstanceCreated'],
  ['Described asset instance created', 'DescribedAssetInstanceCreated'],
  ['Described payment definition created', 'DescribedPaymentDefinitionCreated'],
  ['Member registered', 'MemberRegistered']
];

const subscriptionInfoCorda = [
  ['Asset instance created', 'io.kaleido.kat.states.AssetInstanceCreated'],
  ['Described asset instance created', 'io.kaleido.kat.states.DescribedAssetInstanceCreated'],
  ['Asset instance batch created', 'io.kaleido.kat.states.AssetInstanceBatchCreated'],
  ['Asset instance property set', 'io.kaleido.kat.states.AssetInstancePropertySet']
]

export const ensureEventStreamAndSubscriptions = async () => {
  if(!config.eventStreams.skipSetup) {
    const esMgr = new EventStreamManager(config);
    await esMgr.ensureEventStreamsWithRetry();
  }
};

class EventStreamManager {
  private gatewayPath: string;
  private api: AxiosInstance;
  private streamDetails: any;
  private retryCount: number;
  private retryDelay: number;
  private protocol: string;

  constructor(config: IConfig) {
    const apiURL = new URL(config.apiGateway.apiEndpoint);
    this.gatewayPath = apiURL.pathname.replace(/^\//, '');
    apiURL.pathname = '';
    const creds = `${config.apiGateway.auth?.user??config.appCredentials.user}:${config.apiGateway.auth?.password??config.appCredentials.password}`;
    this.api = axios.create({
      baseURL: apiURL.href,
      headers: {
        Authorization: `Basic ${Buffer.from(creds).toString('base64')}`
      }
    });
    this.api.interceptors.request.use(requestLogger);
    this.api.interceptors.response.use(responseLogger, errorLogger);
    this.retryCount = 20;
    this.retryDelay = 5000;
    this.protocol = config.protocol;
    this.streamDetails = {
      name: config.eventStreams.topic,
      errorHandling: config.eventStreams.config?.errorHandling??'block',
      batchSize: config.eventStreams.config?.batchSize??50,
      batchTimeoutMS: config.eventStreams.config?.batchTimeoutMS??500,
      type: "websocket",
      websocket: {
        topic: config.eventStreams.topic
      }
    }
    if(this.protocol === 'ethereum') {
      // Due to spell error in ethconnect, we do this for now until ethconnect is updated
      this.streamDetails.blockedReryDelaySec = config.eventStreams.config?.blockedRetryDelaySec??30;
    } else {
      this.streamDetails.blockedRetryDelaySec = config.eventStreams.config?.blockedRetryDelaySec??30;
    }
  }

  async ensureEventStreamsWithRetry() {
    for (let i = 1; i <= this.retryCount; i++) {
      try {
        if (i > 1) await sleep(this.retryDelay);
        const stream: IEventStream = await this.ensureEventStream();
        await this.ensureSubscriptions(stream);
        return;
      } catch (err) {
        logger.error(`Attempt ${i} to initialize event streams failed`, err);
      }
    }
    throw new Error("Failed to initialize event streams after retries");
  }

  async ensureEventStream(): Promise<IEventStream> {
    const { data: existingStreams } = await this.api.get('eventstreams');
    let stream = existingStreams.find((s: any) => s.name === this.streamDetails.name);
    if (stream) {
      const { data: patchedStream } = await this.api.patch(`eventstreams/${stream.id}`, this.streamDetails);
      return patchedStream;
    }
    const { data: newStream } = await this.api.post('eventstreams', this.streamDetails);
    return newStream;
  }

  subscriptionInfo() {
    switch(this.protocol) {
      case 'ethereum':
        return subscriptionsInfoEthereum;
      case 'corda':
        return subscriptionInfoCorda;
      default:
        throw new Error("Unsupported protocol.");
    }
  }

  async createSubscription(eventType: string, streamId: string, description: string): Promise<{id: string, name: string}> {
    switch(this.protocol) {
      case 'ethereum':
        return this.api.post(`${this.gatewayPath}/${eventType}/Subscribe`, {
          description,
          name: eventType,
          stream: streamId,
          fromBlock: "0", // Subscribe from the start of the chain
        }).then(r => { logger.info(`Created subscription ${eventType}: ${r.data.id}`); return r.data });
      case 'corda': 
        return this.api.post('subscriptions', {
          name: eventType,
          stream: streamId,
          fromTime: null, // BEGINNING is specified as `null` in Corda event streams
          filter: {
            stateType: eventType,
            stateStatus: "unconsumed",
            relevancyStatus: "all"
          }
        }).then(r => { logger.info(`Created subscription ${eventType}: ${r.data.id}`); return r.data });
      default:
        throw new Error("Unsupported protocol.");
    }
  }



  async ensureSubscriptions(stream: IEventStream) {
    const dbSubscriptions = (await database.retrieveSubscriptions()) || {};
    const { data: existing } = await this.api.get('subscriptions');
    for (const [description, eventName] of this.subscriptionInfo()) {
      const dbEventName = eventName.replace(/\./g, '_');
      let sub = existing.find((s: IEventStreamSubscription) => s.name === eventName && s.stream === stream.id);
      let storedSubId = dbSubscriptions[dbEventName];
      if (!sub || sub.id !== storedSubId) {
        if (sub) {
          logger.info(`Deleting stale subscription that does not match persisted id ${storedSubId}`, sub);
          await this.api.delete(`subscriptions/${sub.id}`);
        }
        const newSub = await this.createSubscription(eventName, stream.id, description);
        dbSubscriptions[dbEventName] = newSub.id;
      } else {
        logger.info(`Subscription ${eventName}: ${sub.id}`);
      }
    }
    await database.upsertSubscriptions(dbSubscriptions);
  }

}
