import Ajv from 'ajv';
import { v4 as uuidV4 } from 'uuid';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as docExchange from '../clients/doc-exchange';
import * as apiGateway from '../clients/api-gateway';
import RequestError from '../lib/request-error';
import { IDBBlockchainData, IEventAssetInstanceCreated, IEventAssetInstancePropertySet } from '../lib/interfaces';

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

export const handleCreateStructuredAssetInstanceRequest = async (author: string, assetDefinitionID: string, description: Object | undefined, content: Object, sync: boolean) => {
  const assetInstanceID = uuidV4();
  let descriptionHash: string | undefined;
  let contentHash: string;
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 400);
  }
  if (!assetDefinition.confirmed) {
    throw new RequestError('Asset definition must be confirmed', 400);
  }
  if (!assetDefinition.contentSchema) {
    throw new RequestError('Unstructured asset instances must be created using multipart/form-data', 400);
  }
  if (assetDefinition.descriptionSchema) {
    if (!description) {
      throw new RequestError('Missing asset description', 400);
    }
    if (!ajv.validate(assetDefinition.descriptionSchema, description)) {
      throw new RequestError('Description does not conform to asset definition schema', 400);
    }
    descriptionHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(description)));
  }
  if (!ajv.validate(assetDefinition.contentSchema, content)) {
    throw new RequestError('Content does not conform to asset definition schema', 400);
  }
  if (assetDefinition.isContentPrivate) {
    contentHash = `0x${utils.getSha256(JSON.stringify(content))}`;
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  if (assetDefinition.isContentUnique && (await database.retrieveAssetInstanceByContentID(assetDefinition.assetDefinitionID, contentHash))?.confirmed) {
    throw new RequestError(`Asset instance content conflict`);
  }
  await database.upsertAssetInstance(assetInstanceID, author, assetDefinitionID, descriptionHash, description, contentHash, content, false, utils.getTimestamp(), undefined);
  if (descriptionHash) {
    await apiGateway.createDescribedAssetInstance(utils.uuidToHex(assetInstanceID), utils.uuidToHex(assetDefinitionID), author, descriptionHash, contentHash, sync); // TODO
  } else {
    await apiGateway.createAssetInstance(utils.uuidToHex(assetInstanceID), utils.uuidToHex(assetDefinitionID), author, contentHash, sync);
  }
  return assetInstanceID;
};

export const handleCreateUnstructuredAssetInstanceRequest = async (author: string, assetDefinitionID: string, description: Object | undefined, content: NodeJS.ReadableStream, contentFileName: string, sync: boolean) => {
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
  }
  if (assetDefinition.isContentPrivate) {
    contentHash = await docExchange.uploadStream(content, utils.getUnstructuredFilePathInDocExchange(assetDefinition.name, assetInstanceID, contentFileName));
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  await database.upsertAssetInstance(assetInstanceID, author, assetDefinitionID, descriptionHash, description, contentHash, undefined, false, utils.getTimestamp(), undefined);
  if (descriptionHash) {
    await apiGateway.createDescribedAssetInstance(utils.uuidToHex(assetInstanceID), assetDefinitionID, author, descriptionHash, contentHash, sync);
  } else {
    await apiGateway.createAssetInstance(utils.uuidToHex(assetInstanceID), assetDefinitionID, author, contentHash, sync);
  }
  return assetInstanceID;
}

export const handleSetAssetInstancePropertyRequest = async (assetInstanceID: string, author: string, key: string, value: string, sync: boolean) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Unknown asset instance', 400);
  }
  if (!assetInstance.confirmed) {
    throw new RequestError('Unconfirmed asset instance', 400);
  }
  if (assetInstance.properties) {
    const authorMetadata = assetInstance.properties[author];
    if (authorMetadata) {
      const currentValue = authorMetadata[key];
      if (currentValue?.confirmed && currentValue.value === value) {
        throw new RequestError('Property already set');
      }
    }
  }
  await database.setAssetInstanceProperty(assetInstanceID, author, key, value, false, utils.getTimestamp(), undefined);
  await apiGateway.setAssetInstanceProperty(utils.uuidToHex(assetInstanceID), author, key, value, sync);
};

export const handleAssetInstanceCreatedEvent = async (event: IEventAssetInstanceCreated, blockchainData: IDBBlockchainData) => {
  const eventAssetInstanceID = utils.hexToUuid(event.assetInstanceID);
  const dbAssetInstance = await database.retrieveAssetInstanceByID(eventAssetInstanceID);
  if (dbAssetInstance !== null && dbAssetInstance.confirmed) {
    throw new Error(`Duplicate asset instance ID`);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(utils.hexToUuid(event.assetDefinitionID));
  if (assetDefinition === null) {
    throw new Error('Uknown asset definition');
  }
  if (!assetDefinition.confirmed) {
    throw new Error('Unconfirmed asset definition');
  }
  if (assetDefinition.isContentUnique) {
    const assetInstanceByContentID = await database.retrieveAssetInstanceByContentID(assetDefinition.assetDefinitionID, event.contentHash);
    if (assetInstanceByContentID !== null && eventAssetInstanceID !== assetInstanceByContentID.assetInstanceID) {
      if (assetInstanceByContentID.confirmed) {
        throw new Error(`Asset instance content conflict ${event.contentHash}`);
      } else {
        await database.markAssetInstanceAsConflict(assetInstanceByContentID.assetInstanceID, Number(event.timestamp));
      }
    }
  }
  let description: Object | undefined = undefined;
  if (assetDefinition.descriptionSchema) {
    if (event.descriptionHash) {
      if (event.descriptionHash === dbAssetInstance?.descriptionHash) {
        description = dbAssetInstance.description;
      } else {
        description = await ipfs.downloadJSON(utils.sha256ToIPFSHash(event.descriptionHash));
        if (!ajv.validate(assetDefinition.descriptionSchema, description)) {
          throw new Error('Description does not conform to schema');
        }
      }
    } else {
      throw new Error('Missing asset instance description');
    }
  }
  let content: Object | undefined = undefined;
  if (assetDefinition.contentSchema) {
    if (event.contentHash === dbAssetInstance?.contentHash) {
      content = dbAssetInstance.content;
    } else if (!assetDefinition.isContentPrivate) {
      content = await ipfs.downloadJSON(event.contentHash);
      if (!ajv.validate(assetDefinition.contentSchema, content)) {
        throw new Error('Content does not conform to schema');
      }
    }
  }
  database.upsertAssetInstance(eventAssetInstanceID, event.author, utils.hexToUuid(event.assetDefinitionID),
    event.descriptionHash, description, event.contentHash, content, true, Number(event.timestamp), blockchainData);
};

export const handleSetAssetInstancePropertyEvent = async (event: IEventAssetInstancePropertySet, blockchainData: IDBBlockchainData) => {
  const eventAssetInstanceID = utils.hexToUuid(event.assetInstanceID);
  const dbAssetInstance = await database.retrieveAssetInstanceByID(eventAssetInstanceID);
  if (dbAssetInstance === null) {
    throw new Error('Uknown asset instance');
  }
  if (!dbAssetInstance.confirmed) {
    throw new Error('Unconfirmed asset instance');
  }
  if (!event.key) {
    throw new Error('Invalid property key');
  }
  await database.setAssetInstanceProperty(eventAssetInstanceID, event.author, event.key, event.value, true, Number(event.timestamp), blockchainData);
};
