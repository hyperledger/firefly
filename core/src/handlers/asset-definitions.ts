import { v4 as uuidV4 } from 'uuid';
import Ajv from 'ajv';
import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import {
  IAPIGatewayAsyncResponse,
  IAPIGatewaySyncResponse,
  IDBBlockchainData,
  IDBAssetDefinition,
  IEventAssetDefinitionCreated
} from '../lib/interfaces';

const ajv = new Ajv();

export const handleGetAssetDefinitionsRequest = (query: object, skip: number, limit: number) => {
  return database.retrieveAssetDefinitions(query, skip, limit);
};

export const handleCountAssetDefinitionsRequest = async (query: object) => {
  return { count: await database.countAssetDefinitions(query) };
};

export const handleGetAssetDefinitionRequest = async (assetDefinitionID: string) => {
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Asset definition not found', 404);
  }
  return assetDefinition;
};

export const handleCreateAssetDefinitionRequest = async (name: string, isContentPrivate: boolean, isContentUnique: boolean,
  author: string, descriptionSchema: Object | undefined, contentSchema: Object | undefined, indexSchema: Object | undefined, sync: boolean) => {
  if (descriptionSchema !== undefined && !ajv.validateSchema(descriptionSchema)) {
    throw new RequestError('Invalid description schema', 400);
  }
  if (contentSchema !== undefined && !ajv.validateSchema(contentSchema)) {
    throw new RequestError('Invalid content schema', 400);
  }
  if (indexSchema !== undefined && !ajv.validateSchema(indexSchema)) {
    throw new RequestError('Invalid index schema', 400);
  }
  if (await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }

  const assetDefinitionID = uuidV4();
  const timestamp = utils.getTimestamp();
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;

  const assetDefinition = {
    assetDefinitionID,
    name,
    isContentPrivate,
    isContentUnique,
    descriptionSchema,
    contentSchema
  };

  const assetDefinitionHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(assetDefinition)));

  apiGatewayResponse = await apiGateway.createAssetDefinition(author, sync, assetDefinitionHash);
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.upsertAssetDefinition({
    assetDefinitionID,
    author,
    name,
    isContentPrivate,
    isContentUnique,
    descriptionSchema,
    assetDefinitionHash,
    contentSchema,
    submitted: timestamp,
    receipt
  });
  return assetDefinitionID;
};

export const handleAssetDefinitionCreatedEvent = async (event: IEventAssetDefinitionCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {
  let assetDefinition = await ipfs.downloadJSON<IDBAssetDefinition>(utils.sha256ToIPFSHash(event.assetDefinitionHash));
  const dbAssetDefinitionByID = await database.retrieveAssetDefinitionByID(assetDefinition.assetDefinitionID);
  if (dbAssetDefinitionByID !== null) {
    if (dbAssetDefinitionByID.transactionHash !== undefined) {
      throw new Error(`Asset definition ID conflict ${assetDefinition.assetDefinitionID}`);
    }
  } else {
    const dbAssetDefinitionByName = await database.retrieveAssetDefinitionByName(event.name);
    if (dbAssetDefinitionByName !== null) {
      if (dbAssetDefinitionByName.transactionHash !== undefined) {
        throw new Error(`Asset definition name conflict ${event.name}`);
      } else {
        await database.markAssetDefinitionAsConflict(assetDefinition.assetDefinitionID, Number(event.timestamp));
      }
    }
  }

  database.upsertAssetDefinition({
    ...assetDefinition,
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  });

  const collectionName = `asset-instance-${assetDefinition.assetDefinitionID}`;
  await database.createCollection(collectionName, [{fields: ['assetInstanceID'], unique: true}]);
};
