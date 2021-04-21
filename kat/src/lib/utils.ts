import { encode, decode } from 'bs58';
import crypto from 'crypto';
import axios, { AxiosRequestConfig } from 'axios';
import { databaseCollectionName, indexes } from './interfaces';
import { parseDN } from 'ldapjs';
import { Logger } from './logging';

export const constants = {
  DATA_DIRECTORY: process.env.DATA_DIRECTORY || '/data',
  LOG_LEVEL: process.env.LOG_LEVEL || 'info',
  CONFIG_FILE_NAME: 'config.json',
  SETTINGS_FILE_NAME: 'settings.json',
  IPFS_TIMEOUT_MS: 15000,
  DEFAULT_PAGINATION_LIMIT: 100,
  EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS: 5,
  DOC_EXCHANGE_ASSET_FOLDER_NAME: 'assets',
  EVENT_STREAM_PING_TIMEOUT_SECONDS: 60,
  ASSET_INSTANCE_TRADE_TIMEOUT_SECONDS: 15,
  TRADE_AUTHORIZATION_TIMEOUT_SECONDS: 10,
  DOCUMENT_EXCHANGE_TRANSFER_TIMEOUT_SECONDS: 15,
  SUBSCRIBE_RETRY_INTERVAL: 5 * 1000,
  APP2APP_BATCH_SIZE: parseInt(<string>process.env.APP2APP_BATCH_SIZE || "100"),
  APP2APP_BATCH_TIMEOUT: parseInt(<string>process.env.APP2APP_BATCH_TIMEOUT || "250"),
  APP2APP_READ_AHEAD: parseInt(<string>process.env.APP2APP_READ_AHEAD || "50"),
  REST_API_CALL_MAX_ATTEMPTS: parseInt(<string>process.env.REST_API_CALL_MAX_ATTEMPTS || "5"),
  REST_API_CALL_RETRY_DELAY_MS: parseInt(<string>process.env.REST_API_CALL_MAX_ATTEMPTS || "500"),
  BATCH_ADD_TIMEOUT_MILLIS: parseInt(<string>process.env.BATCH_ADD_TIMEOUT_MILLIS || '30000'),
  BATCH_TIMEOUT_OVERALL_MILLIS: parseInt(<string>process.env.BATCH_TIMEOUT_OVERALL_MILLIS || '2500'),
  BATCH_TIMEOUT_ARRIVAL_MILLIS: parseInt(<string>process.env.BATCH_TIMEOUT_ARRIVAL_MILLIS || '250'),
  BATCH_MAX_RECORDS: parseInt(<string>process.env.BATCH_MAX_RECORDS || '1000'),
  BATCH_RETRY_INITIAL_DELAY_MILLIS: parseInt(<string>process.env.BATCH_RETRY_INITIAL_DELAY_MILLIS || '100'),
  BATCH_RETRY_MAX_DELAY_MILLIS: parseInt(<string>process.env.BATCH_RETRY_MAX_DELAY_MILLIS || '10000'),
  BATCH_RETRY_MULTIPLIER: parseFloat(<string>process.env.BATCH_RETRY_MULTIPLIER || '2.0'),
};

const log = new Logger('utis.ts');

export const databaseCollectionIndexes: { [name in databaseCollectionName]: indexes } = {
  members: [{ fields: ['address'], unique: true }],
  'asset-definitions': [{ fields: ['assetDefinitionID'], unique: true }],
  'payment-definitions': [{ fields: ['paymentDefinitionID'], unique: true }],
  'payment-instances': [{ fields: ['paymentInstanceID'], unique: true }],
  'batches': [
    { fields: ['batchID'], unique: true }, // Primary key
    { fields: ['type', 'author', 'completed', 'created'] }, // Search index for startup processing, and other queries
    { fields: ['batchHash'] } // To retrieve a batch by its hash, in response to a blockchain event
  ],
  'state': [{ fields: ['key'], unique: true }],
};

const ETHEREUM_ACCOUNT_REGEXP = /^0x[a-fA-F0-9]{40}$/;

const isValidX500Name = (name: string) => {
  try {
    parseDN(name);
  } catch (e) {
    return false;
  }
  return true;
};

export const isAuthorValid = (author: string, protocol: string) => {
  switch (protocol) {
    case 'corda':
      return isValidX500Name(author);
    case 'ethereum':
      return ETHEREUM_ACCOUNT_REGEXP.test(author);
  }
}

export const requestKeys = {
  ASSET_AUTHOR: 'author',
  ASSET_DEFINITION_ID: 'assetDefinitionID',
  ASSET_DESCRIPTION: 'description',
  ASSET_CONTENT: 'content',
  ASSET_IS_CONTENT_PRIVATE: 'isContentPrivate'
};

export const contractEventSignaturesCorda = {
  ASSET_DEFINITION_CREATED: 'io.kaleido.kat.states.AssetDefinitionCreated',
  MEMBER_REGISTERED: 'io.kaleido.kat.states.MemberRegistered',
  DESCRIBED_PAYMENT_DEFINITION_CREATED: 'io.kaleido.kat.states.DescribedPaymentDefinitionCreated',
  PAYMENT_DEFINITION_CREATED: 'io.kaleido.kat.states.PaymentDefinitionCreated',
  DESCRIBED_ASSET_INSTANCE_CREATED: 'io.kaleido.kat.states.DescribedAssetInstanceCreated',
  ASSET_INSTANCE_BATCH_CREATED: 'io.kaleido.kat.states.AssetInstanceBatchCreated',
  ASSET_INSTANCE_CREATED: 'io.kaleido.kat.states.AssetInstanceCreated',
  DESCRIBED_PAYMENT_INSTANCE_CREATED: 'io.kaleido.kat.states.DescribedPaymentInstanceCreated',
  PAYMENT_INSTANCE_CREATED: 'io.kaleido.kat.states.PaymentInstanceCreated',
  ASSET_PROPERTY_SET: 'io.kaleido.kat.states.AssetInstancePropertySet'
}

export const contractEventSignatures = {
  ASSET_DEFINITION_CREATED: 'AssetDefinitionCreated(bytes32,address,uint256)',
  MEMBER_REGISTERED: 'MemberRegistered(address,string,string,string,string,uint256)',
  DESCRIBED_PAYMENT_DEFINITION_CREATED: 'DescribedPaymentDefinitionCreated(bytes32,address,string,bytes32,uint256)',
  PAYMENT_DEFINITION_CREATED: 'PaymentDefinitionCreated(bytes32,address,string,uint256)',
  DESCRIBED_ASSET_INSTANCE_CREATED: 'DescribedAssetInstanceCreated(bytes32,bytes32,address,bytes32,bytes32,uint256)',
  ASSET_INSTANCE_BATCH_CREATED: 'AssetInstanceBatchCreated(bytes32,address,uint256)',
  ASSET_INSTANCE_CREATED: 'AssetInstanceCreated(bytes32,bytes32,address,bytes32,uint256)',
  DESCRIBED_PAYMENT_INSTANCE_CREATED: 'DescribedPaymentInstanceCreated(bytes32,bytes32,address,address,uint256,bytes32,uint256)',
  PAYMENT_INSTANCE_CREATED: 'PaymentInstanceCreated(bytes32,bytes32,address,address,uint256,uint256)',
  ASSET_PROPERTY_SET: 'AssetInstancePropertySet(bytes32,bytes32,address,string,string,uint256)'
};

export const getSha256 = (value: string) => crypto.createHash('sha256').update(value).digest('hex');

export const ipfsHashToSha256 = (hash: string) => '0x' + decode(hash).slice(2).toString('hex');

export const sha256ToIPFSHash = (short: string) => encode(Buffer.from('1220' + short.slice(2), 'hex'));

export const getTimestamp = () => {
  return Math.round(new Date().getTime() / 1000);
};

export const streamToString = (stream: NodeJS.ReadableStream): Promise<string> => {
  const chunks: Buffer[] = [];
  return new Promise((resolve, reject) => {
    stream.on('data', chunk => chunks.push(chunk));
    stream.on('error', reject);
    stream.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
  })
}

export const getUnstructuredFilePathInDocExchange = (assetInstanceID: string) => {
  return `${constants.DOC_EXCHANGE_ASSET_FOLDER_NAME}/${assetInstanceID}`;
};

export const uuidToHex = (uuid: string) => {
  return '0x' + Buffer.from(uuid.replace(/-/g, '')).toString('hex');
};

export const hexToUuid = (hex: string) => {
  const decodedTransferID = Buffer.from(hex.substr(2), 'hex').toString('utf-8');
  return decodedTransferID.substr(0, 8) + '-' + decodedTransferID.substr(8, 4) + '-' +
    decodedTransferID.substr(12, 4) + '-' + decodedTransferID.substr(16, 4) + '-' +
    decodedTransferID.substr(20, 12);
};

export const axiosWithRetry = async (config: AxiosRequestConfig) => {
  let attempts = 0;
  let currentError;
  while (attempts < constants.REST_API_CALL_MAX_ATTEMPTS) {
    try {
      return await axios(config);
    } catch (err) {
      const data = err.response?.data;
      log.error(`${config.method} ${config.url} attempt ${attempts} [${err.response?.status}]`, (data && !data.on) ? data : err.stack)
      if (err.response?.status === 404) {
        throw err;
      } else {
        currentError = err;
        attempts++;
        await new Promise(resolve => setTimeout(resolve, constants.REST_API_CALL_RETRY_DELAY_MS));
      }
    }
  }
  throw currentError;
};

export function getLogger(label: string) {
  return new Logger(label);
}