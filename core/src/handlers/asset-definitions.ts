import { v4 as uuidV4 } from 'uuid';
import Ajv from 'ajv';
import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import indexSchema from '../schemas/indexes.json'
import assetDefinitionSchema from '../schemas/asset-definition.json'

import {
  IAPIGatewayAsyncResponse,
  IAPIGatewaySyncResponse,
  IDBBlockchainData,
  IDBAssetDefinition,
  IEventAssetDefinitionCreated,
  IAssetDefinitionRequest,
  indexes
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
  author: string, descriptionSchema: Object | undefined, contentSchema: Object | undefined, indexes: { fields: string[], unique?: boolean }[] | undefined, sync: boolean) => {
  if (descriptionSchema !== undefined && !ajv.validateSchema(descriptionSchema)) {
    throw new RequestError('Invalid description schema', 400);
  }
  if (contentSchema !== undefined && !ajv.validateSchema(contentSchema)) {
    throw new RequestError('Invalid content schema', 400);
  }
  if (indexes !== undefined && !ajv.validate(indexSchema, indexes)) {
    throw new RequestError('Indexes do not conform to index schema', 400);
  }
  if (await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }

  const assetDefinitionID = uuidV4();
  const timestamp = utils.getTimestamp();
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;

  const assetDefinition: IAssetDefinitionRequest = {
    assetDefinitionID,
    name,
    isContentPrivate,
    isContentUnique,
    descriptionSchema,
    contentSchema,
    indexes
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
    indexes,
    submitted: timestamp,
    receipt
  });
  return assetDefinitionID;
};

export const handleAssetDefinitionCreatedEvent = async (event: IEventAssetDefinitionCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {
  let assetDefinition = await ipfs.downloadJSON<IDBAssetDefinition>(utils.sha256ToIPFSHash(event.assetDefinitionHash));
  if (!ajv.validate(assetDefinitionSchema, assetDefinition)) {
    throw new RequestError(`Invalid asset definition content ${JSON.stringify(ajv.errors)}`, 400);
  }
  const dbAssetDefinitionByID = await database.retrieveAssetDefinitionByID(assetDefinition.assetDefinitionID);
  if (dbAssetDefinitionByID !== null) {
    if (dbAssetDefinitionByID.transactionHash !== undefined) {
      throw new Error(`Asset definition ID conflict ${assetDefinition.assetDefinitionID}`);
    }
  } else {
    const dbAssetDefinitionByName = await database.retrieveAssetDefinitionByName(assetDefinition.name);
    if (dbAssetDefinitionByName !== null) {
      if (dbAssetDefinitionByName.transactionHash !== undefined) {
        throw new Error(`Asset definition name conflict ${dbAssetDefinitionByName.name}`);
      } else {
        await database.markAssetDefinitionAsConflict(dbAssetDefinitionByName.assetDefinitionID, Number(event.timestamp));
      }
    }
  }

  database.upsertAssetDefinition({
    ...assetDefinition,
    author: event.author,
    assetDefinitionHash: event.assetDefinitionHash,
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  });

  const collectionName = `asset-instance-${assetDefinition.assetDefinitionID}`;
  let indexes: indexes = [{ fields: ['assetInstanceID'], unique: true }];
  if (assetDefinition.indexes !== undefined) {
    indexes = indexes.concat(assetDefinition.indexes)
  }
  await database.createCollection(collectionName, indexes);

};
