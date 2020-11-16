import { encode, decode } from 'bs58';
import crypto from 'crypto';

export const constants = {
  DATA_DIRECTORY: process.env.DATA_DIRECTORY || '/data',
  LOG_LEVEL: process.env.LOG_LEVEL || 'info',
  CONFIG_FILE_NAME: 'config.json',
  SETTINGS_FILE_NAME: 'settings.json',
  IPFS_TIMEOUT_MS: 15000,
  DEFAULT_PAGINATION_LIMIT: 100,
  MEMBERS_DATABASE_FILE_NAME: 'members.json',
  ASSET_DEFINITIONS_DATABASE_FILE_NAME: 'asset-definitions.json',
  PAYMENT_DEFINITIONS_DATABASE_FILE_NAME: 'payment-definitions.json',
  ASSET_INSTANCES_DATABASE_FILE_NAME: 'asset-instances.json',
  PAYMENT_INSTANCES_DATABASE_FILE_NAME: 'payment-instances.json',
  EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS: 5,
  DOC_EXCHANGE_ASSET_FOLDER_NAME: 'assets',
  EVENT_STREAM_PING_TIMEOUT_SECONDS: 60,
  ASSET_INSTANCE_TRADE_TIMEOUT_SECONDS: 15,
  DOCUMENT_EXCHANGE_TRANSFER_TIMEOUT_SECONDS: 15
};

export const regexps = {
  ACCOUNT: /^0x[a-fA-F0-9]{40}$/
}

export const requestKeys = {
  ASSET_AUTHOR: 'author',
  ASSET_DEFINITION_ID: 'assetDefinitionID',
  ASSET_DESCRIPTION: 'description',
  ASSET_CONTENT: 'content'
};

export const settingsKeys = {
  PRIVATE_ASSET_INSTANCE_AUTHORIZATION_WEBHOOK: 'privateAssetInstanceAuthorizationWebhook'
}

export const contractEventSignatures = {
  MEMBER_REGISTERED: 'MemberRegistered(address,string,string,string,string,uint256)',
  DESCRIBED_STRUCTURED_ASSET_DEFINITION_CREATED: 'DescribedStructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,bytes32,uint256)',
  DESCRIBED_UNSTRUCTURED_ASSET_DEFINITION_CREATED: 'DescribedUnstructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,uint256)',
  STRUCTURED_ASSET_DEFINITION_CREATED: 'StructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,uint256)',
  UNSTRUCTURED_ASSET_DEFINITION_CREATED: 'UnstructuredAssetDefinitionCreated(uint256,address,string,bool,uint256)',
  DESCRIBED_PAYMENT_DEFINITION_CREATED: 'DescribedPaymentDefinitionCreated(bytes32,address,string,bytes32,uint256)',
  PAYMENT_DEFINITION_CREATED: 'PaymentDefinitionCreated(bytes32,address,string,uint256)',
  DESCRIBED_ASSET_INSTANCE_CREATED: 'DescribedAssetInstanceCreated(bytes32,uint256,address,bytes32,bytes32,uint256)',
  ASSET_INSTANCE_CREATED: 'AssetInstanceCreated(bytes32,bytes32,address,bytes32,uint256)',
  DESCRIBED_PAYMENT_INSTANCE_CREATED: 'DescribedPaymentInstanceCreated(bytes32,bytes32,address,address,uint256,bytes32,uint256)',
  PAYMENT_INSTANCE_CREATED: 'PaymentInstanceCreated(bytes32,bytes32,address,address,uint256,uint256)',  
  ASSET_PROPERTY_SET: 'AssetInstancePropertySet(bytes32,address,string,string,uint256)'
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
