import * as QueryString from 'querystring';
import { URL } from 'url';
import axios, { AxiosInstance } from 'axios';

import { createLogger, LogLevelString } from 'bunyan';
import { promisify } from 'util';
import * as timers from 'timers';
const sleep = promisify(timers.setTimeout);

import * as utils from './utils';
import { config } from './config'
import { IConfig, IEventStream, IEventStreamSubscription } from './interfaces';
const logger = createLogger({ name: 'index.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

/* istanbul ignore next */
const requestLogger = (config: any) => {
  const qs = config.params ? `?${QueryString.stringify(config.params)}` : '';
  logger.info(`--> ${config.method} ${config.baseURL}${config.url}${qs}`);
  logger.debug(config.data);
  return config;
};

/* istanbul ignore next */
const responseLogger = (response: any) => {
  const {config,status,data} = response;
  logger.info(`<-- ${config.method} ${config.url} [${status}]`);
  logger.debug(data);
  return response;
};

/* istanbul ignore next */
const errorLogger = (err: any) => {
  const {config = {}, response = {}} = err;
  const {status,data} = response;
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
  let esMgr = new EventStreamManager(config);
  await esMgr.ensureEventStreamsWithRetry();
};

class EventStreamManager {
  private gatewayPath: string;
  private api: AxiosInstance;
  private streamName: string;
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
    this.streamName = config.eventStreams.topic;
    this.retryCount = 20;
    this.retryDelay = 5000;
    this.protocol = config.protocol;
  }

  async ensureEventStreamsWithRetry() {
    for (let i = 1; i <= this.retryCount; i++) {
      try {
        if (i > 1) await sleep(this.retryDelay);
        const stream: IEventStream = await this.ensureEventStream();
        await this.ensureSubscriptions(stream);
        return;
      } catch(err) {
        logger.error(`Attempt ${i} to initialize event streams failed`, err);
      }
    }
    throw new Error("Failed to initialize event streams after retries");
  }

  async ensureEventStream(): Promise<IEventStream> {
    const streamDetails = {
      name: this.streamName,
      errorHandling: "block",
      blockedReryDelaySec: 30,
      batchTimeoutMS: 500,
      retryTimeoutSec: 0,
      batchSize: 50,
      type: "websocket",
      websocket: {
        topic: this.streamName,
      }
    };
    const {data: existingStreams} = await this.api.get('eventstreams');
    let stream = existingStreams.find((s: any) => s.name === this.streamName);
    if (stream) {
      const {data: patchedStream} = await this.api.patch(`eventstreams/${stream.id}`, streamDetails);
      return patchedStream;
    }
    const {data: newStream} = await this.api.post('eventstreams', streamDetails);
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

  async createSubscription(eventType: string, streamId: string, description: string){
    switch(this.protocol) {
      case 'ethereum':
        return this.api.post(`${this.gatewayPath}/${eventType}/Subscribe`, {
          description,
          name: eventType,
          stream: streamId
        }).then(r => logger.info(`Created subscription ${eventType}: ${r.data.id}`));
      case 'corda': 
        return this.api.post('subscriptions', {
          name: eventType,
          stream: streamId,
          fromTime: null,
          filter: {
            stateType: eventType,
            stateStatus: "unconsumed",
            relevancyStatus: "all"
          }
        }).then(r => logger.info(`Created subscription ${eventType}: ${r.data.id}`));
      default:
        throw new Error("Unsupported protocol.");
    }
  }
  async ensureSubscriptions(stream: IEventStream) {
    const {data: existing} = await this.api.get('subscriptions');
    const promises = [];
    for (const [description, eventName] of this.subscriptionInfo()) {
      let sub = existing.find((s: IEventStreamSubscription) => s.name === eventName && s.stream === stream.id);
      if (!sub) {
        promises.push(this.createSubscription(eventName, stream.id, description));
      } else {
        logger.info(`Subscription ${eventName}: ${sub.id}`);
      }
    }
    await Promise.all(promises);
  }

}
