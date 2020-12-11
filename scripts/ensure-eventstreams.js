const axios = require('axios');
const argv = require('yargs')
  .option('gateway', { alias: 'g', type: 'string' })
  .argv;
const { promisify } = require('util');
const sleep = promisify(require('timers').setTimeout);
const path = require('path');
const fs = require('fs-extra');
const yaml = require('js-yaml');
const jwt = require('jsonwebtoken');

const log = (...stuff) => {
  console.log(`${new Date().toISOString()}: `, ...stuff);
}

const AuthToken = process.env.AUTH_TOKEN;

const requestLogger = (config) => {
  const qs = config.params ? `?${QueryString.stringify(config.params)}` : '';
  log(`--> ${config.method} ${config.baseURL}${config.url}${qs}`);
  log(config.data);
  return config;
};

const responseLogger = (response) => {
  const {config,status,data} = response;
  log(`<-- ${config.method} ${config.baseURL}${config.url} [${status}]`);
  // log(data);
  return response;
};

const errorLogger = (err) => {
  const {config = {}, response = {}} = err;
  const {status,data} = response;
  log(`<-- ${config.method} ${config.baseURL}${config.url} [${status || err}]: ${JSON.stringify(data)}`);
  throw err;
};

const apiKaleido = axios.default.create({
  baseURL: process.env.KALEIDO_URL || 'https://console-us1.kaleido.io/api/v1',
  headers: {
    Authorization: `Bearer ${AuthToken}`
  }
});
apiKaleido.interceptors.request.use(requestLogger);
apiKaleido.interceptors.response.use(responseLogger, errorLogger);

async function execCommand(command, args, options) {
  log.info(`${command} ${args.join(' ')}`);
  const proc = spawn(command, args, options);
  proc.stdout.pipe(process.stdout);
  proc.stderr.pipe(process.stderr);
  await new Promise((resolve, reject) => {
    proc.on('close', code => {
      if (code) reject(new Error(`Exited with ${code}`));
      resolve();
    });
  })
}

async function getMembership() {
  const {data: members} = await apiKaleido.get(`/consortia/${argv.consortia}/memberships?org_name=${argv.member}`);
  if (members.length !== 1) throw new Error(`Did not find member ${argv.member} in consortium ${argv.consortia}`);
  return members[0];
}

async function getMemberNode(membership) {
  const {data: nodes} = await apiKaleido.get(`/consortia/${argv.consortia}/environments/${argv.environment}/nodes?membership_id=${membership._id}`);
  if (nodes.length !== 1) throw new Error(`Did not find member node for member ${membership._id} in env ${argv.environment} of consortium ${argv.consortia}`);
  return nodes[0];
}

async function getKaleidoConnectAPI(membership) {

  const node = await getMemberNode(membership);

  const {data: existingCreds} = await apiKaleido.get(`/consortia/${argv.consortia}/environments/${argv.environment}/appcreds?name=es-script&membership_id=${membership._id}`);
  for (const existing of existingCreds) {
    await apiKaleido.delete(`/consortia/${argv.consortia}/environments/${argv.environment}/appcreds/${existing._id}`);
  }
  const {data: appcred} = await apiKaleido.post(`/consortia/${argv.consortia}/environments/${argv.environment}/appcreds`, {
    name: 'es-script',
    membership_id: membership._id
  });

  let kaleidoConnectAPI = axios.create({
    baseURL: `${node.urls.kaleido_connect}`,
    headers: {
      Authorization: `Basic ${Buffer.from(`${appcred.username}:${appcred.password}`).toString('base64')}`
    }
  });
  kaleidoConnectAPI.interceptors.request.use(requestLogger);
  kaleidoConnectAPI.interceptors.response.use(responseLogger, errorLogger);
  return kaleidoConnectAPI;
}


async function setupEventStreams(kaleidoConnectAPI, membership) {
  const EventStreamName = `${membership.org_name} Event Streams`;
  const {data: eventStreams} = await kaleidoConnectAPI.get('/eventstreams');
  let stream = eventStreams.find(e => e.name === EventStreamName);
  const streamDetails = {
    name: EventStreamName,
    errorHandling: "block",
    blockedReryDelaySec: 30,
    batchTimeoutMS: 500,
    retryTimeoutSec: 0,
    batchSize: 50,
    type: "websocket",
    websocket: {
      topic: argv.topic || 'synaptic'
    }
  };
  if (!stream) {
    stream = await kaleidoConnectAPI.post('/eventstreams', streamDetails).then(r => r.data);
  } else {
    stream = await kaleidoConnectAPI.patch(`/eventstreams/${stream.id}`, streamDetails).then(r => r.data);
  }
  log(`Event stream "${EventStreamName}": ${stream}`);
  return stream;
}

async function ensureSubscriptions(kaleidoConnectAPI, stream) {
  let url = `/subscriptions`;
  const {data: existing} = await kaleidoConnectAPI.get(url);
  for ([description, eventName] of [
    ['Asset instance created', 'AssetInstanceCreated'],
    ['Payment instance created', 'PaymentInstanceCreated'],
    ['Payment definition created', 'PaymentDefinitionCreated'],
    ['Described unstructured asset definition created', 'DescribedUnstructuredAssetDefinitionCreated'],
    ['Unstructured asset definition created', 'UnstructuredAssetDefinitionCreated'],
    ['Structured asset definition created', 'StructuredAssetDefinitionCreated'],
    ['Asset instance property set', 'AssetInstancePropertySet'],
    ['Described payment instance created', 'DescribedPaymentInstanceCreated'],
    ['Described asset instance created', 'DescribedAssetInstanceCreated'],
    ['Described payment definition created', 'DescribedPaymentDefinitionCreated'],
    ['Described structured asset definition created', 'DescribedStructuredAssetDefinitionCreated'],
    ['Member registered', 'MemberRegistered']
  ]) {
    let sub = existing.find(s => s.name === eventName);
    if (!sub) {
      sub = await kaleidoConnectAPI.post(`/instances/${argv.gateway}/${eventName}/Subscribe`, {
        description,
        name: eventName,
        stream: stream.id,
      }).then(r => r.data);
    }
    log(`Subscription ${eventName}: ${sub.id}`);
  }
}

async function run() {
  const membership = await getMembership();
  const kaleidoConnectAPI = await getKaleidoConnectAPI(membership);
  const stream = await setupEventStreams(kaleidoConnectAPI, membership);
  ensureSubscriptions(kaleidoConnectAPI, stream)

}

if (!AuthToken) {
  throw new Error('Must set the auth token with environment variable "AUTH_TOKEN"');
} else {
  run()
    .catch(err => {
      console.error(err.stack);
      process.exit(1);
    });
}