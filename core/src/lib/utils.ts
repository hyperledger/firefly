import { encode, decode } from 'bs58';
import crypto from 'crypto';

export const constants = {
  DATA_DIRECTORY: process.env.DATA_DIRECTORY || '/data',
  CONFIG_FILE_NAME: 'config.json',
  IPFS_TIMEOUT_MS: 15000,
  DEFAULT_PAGINATION_LIMIT: 100,
  MEMBERS_DATABASE_FILE_NAME: 'members.json',
  ASSET_DEFINITIONS_DATABASE_FILE_NAME: 'asset-definitions.json',
  PAYMENT_DEFINITIONS_DATABASE_FILE_NAME: 'payment-definitions.json',
  ASSET_INSTANCES_DATABASE_FILE_NAME: 'asset-instances.json',
  EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS: 5
};

export const requestKeys = {
  ASSET_AUTHOR: 'author',
  ASSET_DEFINITION_ID: 'assetDefinitionID',
  ASSET_DESCRIPTION: 'description',
  ASSET_CONTENT: 'content'
};

export const contractEventSignatures = {
  MEMBER_REGISTERED: 'MemberRegistered(address,string,string,string,uint256)',
  DESCRIBED_STRUCTURED_ASSET_DEFINITION_CREATED: 'DescribedStructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,bytes32,uint256)',
  DESCRIBED_UNSTRUCTURED_ASSET_DEFINITION_CREATED: 'DescribedUnstructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,uint256)',
  STRUCTURED_ASSET_DEFINITION_CREATED: 'StructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,uint256)',
  UNSTRUCTURED_ASSET_DEFINITION_CREATED: 'UnstructuredAssetDefinitionCreated(uint256,address,string,bool,uint256)',
  DESCRIBED_PAYMENT_DEFINITION_CREATED: 'DescribedPaymentDefinitionCreated(uint256,address,string,bytes32,uint256,uint256)',
  PAYMENT_DEFINITION_CREATED: 'PaymentDefinitionCreated(uint256,address,string,uint256,uint256)'
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
