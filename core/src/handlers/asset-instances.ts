import Ajv from 'ajv';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as docExchange from '../clients/doc-exchange';
import * as apiGateway from '../clients/api-gateway';
import RequestError from '../lib/request-error';

const ajv = new Ajv();

export const handleGetAssetInstancesRequest = (skip: number, limit: number) => {
  return database.retrieveAssetInstances(skip, limit);
};

export const handleGetAssetInstanceRequest = async (assetInstanceID: number) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Asset instance not found', 404);
  }
  return assetInstance;
};

export const handleCreateStructuredAssetInstanceRequest = async (author: string, assetDefinitionID: number, description: Object | undefined, content: Object, sync: boolean) => {
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
  if(!ajv.validate(assetDefinition.contentSchema, content)) {
    throw new RequestError('Content does not conform to asset definition schema', 400);
  }
  if(assetDefinition.isContentPrivate) {
    contentHash = utils.getSha256(JSON.stringify(content));
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  await apiGateway.createAssetInstance(assetDefinitionID, author, contentHash, sync);
  await database.upsertAssetInstance(author, assetDefinitionID, description, contentHash, content, 'authored', utils.getTimestamp());
};

export const handleCreateUnstructuredAssetInstanceRequest = async (author: string, assetDefinitionID: number, description: Object | undefined, content: NodeJS.ReadableStream, sync: boolean) => {
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
  if(assetDefinition.isContentPrivate) {
    contentHash = await docExchange.uploadStream(content, 'PATH'); // TODO : PATH !
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  await apiGateway.createAssetInstance(assetDefinitionID, author, contentHash, sync);
  await database.upsertAssetInstance(author, assetDefinitionID, description, contentHash, undefined, 'authored', utils.getTimestamp());
}
