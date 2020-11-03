import Ajv from 'ajv';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as docExchange from '../clients/doc-exchange';
import RequestError from '../lib/request-error';

const ajv = new Ajv();

export const handleGetAssetInstancesRequest = (skip: number, limit: number) => {
  return database.retrieveAssetInstances(skip, limit);
};

export const handleCreateAssetInstanceRequest = async (author: string, assetDefinitionID: number, description: Object | undefined, content: Object | undefined, stream?: NodeJS.ReadableStream) => {
  let contentHash: string;
  const { assetDefinition, descriptionHash } = await commonTasks(author, assetDefinitionID, description);
  if (assetDefinition.contentSchema) {
    if (!ajv.validate(assetDefinition.contentSchema, content)) {
      throw new RequestError('Content does not conform to schema', 400);
    }
    contentHash = utils.getSha256(JSON.stringify(content));
  } else {
    if (assetDefinition.isContentPrivate) {
      contentHash = await docExchange.uploadString(JSON.stringify(content), 'XXXX'); // TODO !!!!!
    } else {
      contentHash = await ipfs.uploadString(JSON.stringify(content));
    }
  }
  // TODO run transaction (described / not described)
  database.upsertAssetInstance(author, assetDefinitionID, description, contentHash, content, 'authored', utils.getTimestamp());
};

export const handleCreateAssetInstanceMultiPartRequest = async (author: string, assetDefinitionID: number, description: Object | undefined, contentStream: NodeJS.ReadableStream) => {
  let contentHash: string;
  let content: Object | undefined;
  const { assetDefinition, descriptionHash } = await commonTasks(author, assetDefinitionID, description);
  if (assetDefinition.contentSchema) {
    content = JSON.parse(await utils.streamToString(contentStream));
    if (!ajv.validate(assetDefinition.contentSchema, content)) {
      throw new RequestError('Content does not conform to schema', 400);
    }
    contentHash = utils.getSha256(JSON.stringify(content));
  } else {
    if (assetDefinition.isContentPrivate) {
      contentHash = await docExchange.uploadStream(contentStream, 'XXXX'); // TODO !!!!!
    } else {
      contentHash = await ipfs.uploadStream(contentStream);
    }
  }
  // TODO run transaction (described / not described)
  database.upsertAssetInstance(author, assetDefinitionID, description, contentHash, content, 'authored', utils.getTimestamp());
}

const commonTasks = async (author: string, assetDefinitionID: number, description: Object | undefined) => {
  let descriptionHash: string | undefined;
  const member = await database.retrieveMember(author);
  if (member === null) {
    throw new RequestError('Unknown author', 400);
  } else if (!member.owned) {
    throw new RequestError('Unauthorized author', 403);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition');
  }
  if (assetDefinition.descriptionSchema) {
    if (!ajv.validate(assetDefinition.descriptionSchema, description)) {
      throw new RequestError('Description does not conform to schema', 400);
    }
    descriptionHash = await ipfs.uploadString(JSON.stringify(description));
  } else if (description) {
    throw new RequestError('Asset cannot have description', 400);
  }
  return { assetDefinition, descriptionHash };
};
