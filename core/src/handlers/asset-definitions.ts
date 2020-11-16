import { v4 as uuidV4 } from 'uuid';
import Ajv from 'ajv';
import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IDBBlockchainData, IEventAssetDefinitionCreated } from '../lib/interfaces';

const ajv = new Ajv();

export const handleGetAssetDefinitionsRequest = (query: object, skip: number, limit: number) => {
  return database.retrieveAssetDefinitions(query, skip, limit);
};

export const handleGetAssetDefinitionRequest = async (assetDefinitionID: string) => {
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Asset definition not found', 404);
  }
  return assetDefinition;
};

export const handleCreateAssetDefinitionRequest = async (name: string, isContentPrivate: boolean, isContentUnique: boolean,
  author: string, descriptionSchema: Object | undefined, contentSchema: Object | undefined, sync: boolean) => {
  if (descriptionSchema !== undefined && !ajv.validateSchema(descriptionSchema)) {
    throw new RequestError('Invalid description schema', 400);
  }
  if (contentSchema !== undefined && !ajv.validateSchema(contentSchema)) {
    throw new RequestError('Invalid content schema', 400);
  }
  if (await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }
  const assetDefinitionID = uuidV4();
  const timestamp = utils.getTimestamp();
  let descriptionSchemaHash: string | undefined;
  let contentSchemaHash: string | undefined;
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
  if (descriptionSchema) {
    descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    if (contentSchema) {
      contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
      apiGatewayResponse = await apiGateway.createDescribedStructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, isContentUnique,
        descriptionSchemaHash, contentSchemaHash, sync);
    } else {
      apiGatewayResponse = await apiGateway.createDescribedUnstructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, isContentUnique,
        descriptionSchemaHash, sync);
    }
  } else if (contentSchema) {
    contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
    apiGatewayResponse = await apiGateway.createStructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, isContentUnique, contentSchemaHash, sync);
  } else {
    apiGatewayResponse = await apiGateway.createUnstructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, isContentUnique, sync);
  }
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.upsertAssetDefinition({
    assetDefinitionID,
    author,
    name,
    isContentPrivate,
    isContentUnique,
    descriptionSchemaHash,
    descriptionSchema,
    contentSchemaHash,
    contentSchema,
    submitted: timestamp,
    receipt
  });
  return assetDefinitionID;
};

export const handleAssetDefinitionCreatedEvent = async (event: IEventAssetDefinitionCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {
  const assetDefinitionID = utils.hexToUuid(event.assetDefinitionID);
  const dbAssetDefinitionByID = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (dbAssetDefinitionByID !== null) {
    if (dbAssetDefinitionByID.transactionHash !== undefined) {
      throw new Error(`Asset definition ID conflict ${assetDefinitionID}`);
    }
  } else {
    const dbAssetDefinitionByName = await database.retrieveAssetDefinitionByName(event.name);
    if (dbAssetDefinitionByName !== null) {
      if (dbAssetDefinitionByName.transactionHash !== undefined) {
        throw new Error(`Asset definition name conflict ${event.name}`);
      } else {
        await database.markAssetDefinitionAsConflict(assetDefinitionID, Number(event.timestamp));
      }
    }
  }
  let descriptionSchema;
  if (event.descriptionSchemaHash) {
    if (event.descriptionSchemaHash === dbAssetDefinitionByID?.descriptionSchemaHash) {
      descriptionSchema = dbAssetDefinitionByID?.descriptionSchema
    } else {
      descriptionSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.descriptionSchemaHash));
    }
  }
  let contentSchema;
  if (event.contentSchemaHash) {
    if (event.contentSchemaHash === dbAssetDefinitionByID?.contentSchemaHash) {
      contentSchema = dbAssetDefinitionByID.contentSchema;
    } else {
      contentSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.contentSchemaHash));
    }
  }
  database.upsertAssetDefinition({
    assetDefinitionID,
    name: event.name,
    author: event.author,
    isContentPrivate: event.isContentPrivate,
    isContentUnique: event.isContentUnique,
    descriptionSchemaHash: event.descriptionSchemaHash,
    descriptionSchema,
    contentSchemaHash: event.contentSchemaHash,
    contentSchema,
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  });
};
