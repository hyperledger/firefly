import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IEventAssetDefinitionCreated } from '../lib/interfaces';

export const handleGetAssetDefinitionsRequest = (skip: number, limit: number) => {
  return database.retrieveAssetDefinitions(skip, limit);
};

export const handleGetAssetDefinitionRequest = async (assetDefinitionID: number) => {
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if(assetDefinition === null) {
    throw new RequestError('Asset definition not found', 404);
  }
  return assetDefinition;
};

export const handleCreateAssetDefinitionRequest = async (name: string, isContentPrivate: boolean, author: string, descriptionSchema?: Object, contentSchema?: Object) => {
  
  if(await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }

  if (descriptionSchema) {
    const descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    if (contentSchema) {
      // Described - structured
      const contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
      await apiGateway.createDescribedStructuredAssetDefinition(name, author, isContentPrivate, descriptionSchemaHash, contentSchemaHash);
      await database.upsertAssetDefinition(name, author, isContentPrivate, descriptionSchema, contentSchema, utils.getTimestamp(), false);
    } else {
      // Described - unstructured
      await apiGateway.createDescribedUnstructuredAssetDefinition(name, author, isContentPrivate, descriptionSchemaHash);
      await database.upsertAssetDefinition(name, author, isContentPrivate, descriptionSchema, undefined, utils.getTimestamp(), false);
    }
  } else if (contentSchema) {
    // Structured
    const contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
    await apiGateway.createStructuredAssetDefinition(name, author, isContentPrivate, contentSchemaHash);
    await database.upsertAssetDefinition(name, author, isContentPrivate, undefined, contentSchema, utils.getTimestamp(), false);
  } else {
    // Unstructured
    await apiGateway.createUnstructuredAssetDefinition(name, author, isContentPrivate);
    await database.upsertAssetDefinition(name, author, isContentPrivate, undefined, undefined, utils.getTimestamp(), false);
  }

};

export const handleAssetDefinitionCreatedEvent = async (event: IEventAssetDefinitionCreated) => {
  if(await database.retrieveAssetDefinitionByName(event.name) !== null) {
    await database.confirmAssetDefinition(event.name, Number(event.timestamp), Number(event.assetDefinitionID));
  } else {
    let descriptionSchema;
    if(event.descriptionSchemaHash) {
      descriptionSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.descriptionSchemaHash));
    }
    let contentSchema;
    if(event.contentSchemaHash) {
      contentSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.contentSchemaHash));
    }
    database.upsertAssetDefinition(event.name, event.author, event.isContentPrivate, descriptionSchema, contentSchema, Number(event.timestamp), true, Number(event.assetDefinitionID));
  }
};
