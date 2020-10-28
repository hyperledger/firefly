import { encode, decode } from 'bs58';

export const constants = {
  DATA_DIRECTORY: process.env.DATA_DIRECTORY || '/data',
  CONFIG_FILE_NAME: 'config.json',
  IPFS_TIMEOUT_MS: 15000,
  DEFAULT_PAGINATION_LIMIT: 100,
  MEMBERS_DATABASE_FILE_NAME: 'members.json',
  ASSET_DEFINITIONS_DATABASE_FILE_NAME: 'asset-definitions.json',
  EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS: 5
};

export const contractEventSignatures = {
  MEMBER_REGISTERED: 'MemberRegistered(address,string,string,string,uint256)',
  DESCRIBED_STRUCTURED_ASSET_DEFINITION_CREATED: 'DescribedStructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,bytes32,uint256)',
  DESCRIBED_UNSTRUCTURED_ASSET_DEFINITION_CREATED: 'DescribedUnstructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,uint256)',
  STRUCTURED_ASSET_DEFINITION_CREATED: 'StructuredAssetDefinitionCreated(uint256,address,string,bool,bytes32,uint256)',
  UNSTRUCTURED_ASSET_DEFINITION_CREATED: 'UnstructuredAssetDefinitionCreated(uint256,address,string,bool,uint256)'
};

export const ipfsHashToSha256 = (hash: string) => '0x' + decode(hash).slice(2).toString('hex');

export const sha256ToIPFSHash = (short: string) => encode(Buffer.from('1220' + short.slice(2), 'hex'));

export const getTimestamp = () => {
  return Math.round(new Date().getTime() / 1000);
};