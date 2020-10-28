import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IEventAssetDefinitionCreated } from '../lib/interfaces';

export const handleGetAssetDefinitionsRequest = (skip: number, limit: number) => {
  return database.retrieveAssetDefinitions(skip, limit);
};

export const handleCreateAssetDefinitionRequest = async (name: string, isContentPrivate: boolean, author: string, sync: boolean, descriptionSchema?: Object, contentSchema?: Object) => {
  
  if(await database.retrieveAssetDefinition(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }

  if (descriptionSchema) {
    const descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    if (contentSchema) {
      // Described - structured
      const contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
      await apiGateway.createDescribedStructuredAssetDefinition(name, author, isContentPrivate, descriptionSchemaHash, contentSchemaHash, sync);
      if(!sync) {
        await database.insertAssetDefinition(name, author, isContentPrivate, descriptionSchema, contentSchema, utils.getTimestamp());
      }
    } else {
      // Described - unstructured
      await apiGateway.createDescribedUnstructuredAssetDefinition(name, author, isContentPrivate, descriptionSchemaHash, sync);
      if(!sync) {
        await database.insertAssetDefinition(name, author, isContentPrivate, descriptionSchema, undefined, utils.getTimestamp());
      }
    }
  } else if (contentSchema) {
    // Structured
    const contentSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(contentSchema)));
    await apiGateway.createStructuredAssetDefinition(name, author, isContentPrivate, contentSchemaHash, sync);
    await database.insertAssetDefinition(name, author, isContentPrivate, undefined, contentSchema, utils.getTimestamp());
  } else {
    // Unstructured
    await apiGateway.createUnstructuredAssetDefinition(name, author, isContentPrivate, sync);
    await database.insertAssetDefinition(name, author, isContentPrivate, undefined, undefined, utils.getTimestamp());
  }

};

export const handleAssetDefinitionCreatedEvent = async (event: IEventAssetDefinitionCreated) => {
  if(await database.retrieveAssetDefinition(event.name) !== null) {
    await database.confirmAssetDefinition(event.name, Number(event.timestamp), Number(event.assetDefinitionID));
  } else {
    let descriptionSchema;
    if(event.descriptionSchemaHash) {
      descriptionSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.descriptionSchemaHash));
    }
    let contentSchema;
    if(event.contentSchemaHash) {
      await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.contentSchemaHash));
    }
    database.insertAssetDefinition(event.name, event.author, event.isContentPrivate, descriptionSchema, contentSchema, Number(event.timestamp), Number(event.assetDefinitionID));
  }
};
