import Ajv from 'ajv';
import { v4 as uuidV4 } from 'uuid';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as docExchange from '../clients/doc-exchange';
import * as apiGateway from '../clients/api-gateway';
import RequestError from '../lib/request-error';
import { IAssetInstanceCreated } from '../lib/interfaces';

const ajv = new Ajv();

export const handleGetAssetInstancesRequest = (skip: number, limit: number) => {
  return database.retrieveAssetInstances(skip, limit);
};

export const handleGetAssetInstanceRequest = async (assetInstanceID: string) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Asset instance not found', 404);
  }
  return assetInstance;
};

export const handleCreateStructuredAssetInstanceRequest = async (author: string, assetDefinitionID: number, description: Object | undefined, content: Object, sync: boolean) => {
  const assetInstanceID = uuidV4();
  let descriptionHash: string | undefined;
  let contentHash: string;
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 400);
  }
  if (!assetDefinition.contentSchema) {
    throw new RequestError('Unstructured asset instances must be created using multipart/form-data', 400);
  }
  if (assetDefinition.descriptionSchema) {
    if (!ajv.validate(assetDefinition.descriptionSchema, description)) {
      throw new RequestError('Description does not conform to asset definition schema', 400);
    }
    descriptionHash = await ipfs.uploadString(JSON.stringify(description));
  } else if (description) {
    throw new RequestError('Asset cannot have description', 400);
  }
  if (!ajv.validate(assetDefinition.contentSchema, content)) {
    throw new RequestError('Content does not conform to asset definition schema', 400);
  }
  if (assetDefinition.isContentPrivate) {
    contentHash = utils.getSha256(JSON.stringify(content));
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  if(descriptionHash) {
    await apiGateway.createDescribedAssetInstance(utils.uuidToHex(assetInstanceID), assetDefinitionID, author, descriptionHash, contentHash, sync);
  } else {
    await apiGateway.createAssetInstance(utils.uuidToHex(assetInstanceID), assetDefinitionID, author, contentHash, sync);
  }
  await database.upsertAssetInstance(assetInstanceID, author, assetDefinitionID, description, contentHash, content, false, utils.getTimestamp());
};

export const handleCreateUnstructuredAssetInstanceRequest = async (author: string, assetDefinitionID: number, description: Object | undefined, content: NodeJS.ReadableStream, contentFileName: string, sync: boolean) => {
  const assetInstanceID = uuidV4();
  let descriptionHash: string | undefined;
  let contentHash: string;
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 400);
  }
  if (assetDefinition.contentSchema) {
    throw new RequestError('Structured asset instances must be created using JSON', 400);
  }
  if (assetDefinition.descriptionSchema) {
    if (!ajv.validate(assetDefinition.descriptionSchema, description)) {
      throw new RequestError('Description does not conform to asset definition schema', 400);
    }
    descriptionHash = await ipfs.uploadString(JSON.stringify(description));
  } else if (description) {
    throw new RequestError('Asset cannot have description', 400);
  }
  if (assetDefinition.isContentPrivate) {
    contentHash = await docExchange.uploadStream(content, utils.getUnstructuredFilePathInDocExchange(assetDefinition.name, assetInstanceID, contentFileName));
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  if(descriptionHash) {
    await apiGateway.createDescribedAssetInstance(utils.uuidToHex(assetInstanceID), assetDefinitionID, author, descriptionHash, contentHash, sync);
  } else {
    await apiGateway.createAssetInstance(utils.uuidToHex(assetInstanceID), assetDefinitionID, author, contentHash, sync);
  }
  await database.upsertAssetInstance(assetInstanceID, author, assetDefinitionID, description, contentHash, undefined, false, utils.getTimestamp());
}

export const handleAssetInstanceCreatedEvent = async (event: IAssetInstanceCreated) => {
  const assetInstance = await database.retrieveAssetInstanceByID(event.assetDefinitionID);
   if(assetInstance !== null) {
    if(assetInstance.confirmed) {
      throw new Error(`Duplicate asset instance ID`);
    } else {
      database.confirmAssetInstance(assetInstance.assetInstanceID, Number(event.timestamp));
    }
  } else {
    const assetDefinition = await database.retrieveAssetDefinitionByID(Number(event.assetDefinitionID));
    if (assetDefinition === null) {
      throw new Error('Uknown asset definition');
    }
    let description: Object | undefined = undefined;
    if (assetDefinition.descriptionSchema) {
      if (event.descriptionHash) {
        description = await ipfs.downloadJSON(event.descriptionHash);
        if (!ajv.validate(assetDefinition.descriptionSchema, description)) {
          throw new Error('Description does not conform to schema');
        }
      } else {
        throw new Error('Missing description');
      }
    }
    let content: Object | undefined = undefined;
    if (assetDefinition.contentSchema && !assetDefinition.isContentPrivate) {
      content = await ipfs.downloadJSON(event.contentHash);
      if (!ajv.validate(assetDefinition.contentSchema, content)) {
        throw new Error('Content does not conform to schema');
      }
    }
    database.upsertAssetInstance(utils.hexToUuid(event.assetInstanceID), event.author, Number(event.assetDefinitionID), description, event.contentHash, content, true, Number(event.timestamp));
  }
};
