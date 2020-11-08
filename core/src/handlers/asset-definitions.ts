import { v4 as uuidV4 } from 'uuid';
import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IEventAssetDefinitionCreated } from '../lib/interfaces';

export const handleGetAssetDefinitionsRequest = (skip: number, limit: number) => {
  return database.retrieveAssetDefinitions(skip, limit);
};

export const handleGetAssetDefinitionRequest = async (assetDefinitionID: string) => {
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Asset definition not found', 404);
  }
  return assetDefinition;
};

export const handleCreateAssetDefinitionRequest = async (name: string, isContentPrivate: boolean, author: string, descriptionSchema: Object | undefined, contentSchema: Object | undefined, sync: boolean) => {
  if (await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }
  const assetDefinitionID = uuidV4();

  if (descriptionSchema) {
    const descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    if (contentSchema) {
      const contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
      await database.upsertAssetDefinition(assetDefinitionID, name, author, isContentPrivate, descriptionSchemaHash, descriptionSchema, contentSchemaHash, contentSchema, utils.getTimestamp(), false);
      await apiGateway.createDescribedStructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, descriptionSchemaHash, contentSchemaHash, sync);
    } else {
      await database.upsertAssetDefinition(assetDefinitionID, name, author, isContentPrivate, descriptionSchemaHash, descriptionSchema, undefined, undefined, utils.getTimestamp(), false);
      await apiGateway.createDescribedUnstructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, descriptionSchemaHash, sync);
    }
  } else if (contentSchema) {
    const contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
    await database.upsertAssetDefinition(assetDefinitionID, name, author, isContentPrivate, undefined, undefined, contentSchemaHash, contentSchema, utils.getTimestamp(), false);
    await apiGateway.createStructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, contentSchemaHash, sync);
  } else {
    await database.upsertAssetDefinition(assetDefinitionID, name, author, isContentPrivate, undefined, undefined, undefined, undefined, utils.getTimestamp(), false);
    await apiGateway.createUnstructuredAssetDefinition(assetDefinitionID, name, author, isContentPrivate, sync);
  }
  return assetDefinitionID;
};

export const handleAssetDefinitionCreatedEvent = async (event: IEventAssetDefinitionCreated) => {
  const assetDefinitionID = utils.hexToUuid(event.assetDefinitionID);
  const dbAssetDefinitionByID = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (dbAssetDefinitionByID !== null) {
    if (dbAssetDefinitionByID.confirmed) {
      throw new Error(`Asset definition ID conflict ${assetDefinitionID}`);
    }
  } else {
    const dbAssetDefinitionByName = await database.retrieveAssetDefinitionByName(event.name);
    if (dbAssetDefinitionByName !== null) {
      if (dbAssetDefinitionByName.confirmed) {
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
    if(event.contentSchemaHash === dbAssetDefinitionByID?.contentSchemaHash) {
      contentSchema = dbAssetDefinitionByID.contentSchema;
    } else {
      contentSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.contentSchemaHash));
    }
  }
  database.upsertAssetDefinition(assetDefinitionID, event.name, event.author, event.isContentPrivate, event.descriptionSchemaHash, descriptionSchema, event.contentSchemaHash, contentSchema, Number(event.timestamp), true);
};
