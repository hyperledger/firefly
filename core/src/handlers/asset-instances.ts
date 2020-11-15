import Ajv from 'ajv';
import { v4 as uuidV4 } from 'uuid';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as docExchange from '../clients/doc-exchange';
import * as apiGateway from '../clients/api-gateway';
import * as app2app from '../clients/app2app';
import RequestError from '../lib/request-error';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IAssetTradeRequest, IDBBlockchainData, IEventAssetInstanceCreated, IEventAssetInstancePropertySet } from '../lib/interfaces';

const ajv = new Ajv();

export const handleGetAssetInstancesRequest = (skip: number, limit: number) => {
  return database.retrieveAssetInstances(skip, limit);
};

export const handleGetAssetInstanceRequest = async (assetInstanceID: string, content: boolean) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Asset instance not found', 404);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetInstance.assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Asset definition not found', 500);
  }
  if (content) {
    if (assetDefinition.contentSchemaHash) {
      return assetInstance.content;
    } else {
      try {
        return await docExchange.downloadStream(utils.getUnstructuredFilePathInDocExchange(assetInstance.assetInstanceID));
      } catch (err) {
        if (err.response?.status === 404) {
          throw new RequestError('Asset instance content not present in off-chain storage', 404);
        } else {
          throw new RequestError(`Failed to obtain asset content from off-chain storage. ${err}`, 500);
        }
      }
    }
  }
  return assetInstance;
};

export const handleCreateStructuredAssetInstanceRequest = async (author: string, assetDefinitionID: string, description: Object | undefined, content: Object, sync: boolean) => {
  let descriptionHash: string | undefined;
  let contentHash: string;
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 400);
  }
  if (assetDefinition.transactionHash === undefined) {
    throw new RequestError('Asset definition transaction must be mined', 400);
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
  if (assetDefinition.isContentUnique && (await database.retrieveAssetInstanceByDefinitionIDAndContentHash(assetDefinition.assetDefinitionID, contentHash)) !== null) {
    throw new RequestError(`Asset instance content conflict`);
  }
  const assetInstanceID = uuidV4();
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
  const timestamp = utils.getTimestamp();
  if (descriptionHash) {
    apiGatewayResponse = await apiGateway.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, sync);
  } else {
    apiGatewayResponse = await apiGateway.createAssetInstance(assetInstanceID, assetDefinitionID, author, contentHash, sync);
  }
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.upsertAssetInstance({
    assetInstanceID,
    author,
    assetDefinitionID,
    descriptionHash,
    description,
    contentHash,
    content,
    submitted: timestamp,
    receipt
  });
  return assetInstanceID;
};

export const handleCreateUnstructuredAssetInstanceRequest = async (author: string, assetDefinitionID: string, description: Object | undefined, content: NodeJS.ReadableStream, _contentFileName: string, sync: boolean) => {
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
  const assetInstanceID = uuidV4();
  if (assetDefinition.isContentPrivate) {
    contentHash = `0x${await docExchange.uploadStream(content, utils.getUnstructuredFilePathInDocExchange(assetInstanceID))}`;
  } else {
    contentHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(content)));
  }
  if (assetDefinition.isContentUnique && (await database.retrieveAssetInstanceByDefinitionIDAndContentHash(assetDefinitionID, contentHash)) !== null) {
    throw new RequestError('Asset instance content conflict', 409);
  }
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
  const timestamp = utils.getTimestamp();
  if (descriptionHash) {
    apiGatewayResponse = await apiGateway.createDescribedAssetInstance(utils.uuidToHex(assetInstanceID),
      utils.uuidToHex(assetDefinitionID), author, descriptionHash, contentHash, sync);
  } else {
    apiGatewayResponse = await apiGateway.createAssetInstance(utils.uuidToHex(assetInstanceID), utils.uuidToHex(assetDefinitionID), author, contentHash, sync);
  }
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.upsertAssetInstance({
    assetInstanceID,
    author,
    assetDefinitionID,
    descriptionHash,
    description,
    contentHash,
    submitted: timestamp,
    receipt
  });
  return assetInstanceID;
}

export const handleSetAssetInstancePropertyRequest = async (assetInstanceID: string, author: string, key: string, value: string, sync: boolean) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Unknown asset instance', 400);
  }
  if (assetInstance.transactionHash === undefined) {
    throw new RequestError('Asset instance transaction must be mined', 400);
  }
  if (assetInstance.properties) {
    const authorMetadata = assetInstance.properties[author];
    if (authorMetadata) {
      const valueData = authorMetadata[key];
      if (valueData?.value === value && valueData.history !== undefined) {
        const keys = Object.keys(valueData.history);
        const lastConfirmedValue = valueData.history[keys[keys.length - 1]];
        if (lastConfirmedValue.value === value) {
          throw new RequestError('Property already set');
        }
      }
    }
  }
  const timestamp = utils.getTimestamp();
  const apiGatewayResponse = await apiGateway.setAssetInstanceProperty(utils.uuidToHex(assetInstanceID), author, key, value, sync);
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.setSubmittedAssetInstanceProperty(assetInstanceID, author, key, value, timestamp, receipt);
};

export const handleAssetInstanceCreatedEvent = async (event: IEventAssetInstanceCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {
  const eventAssetInstanceID = utils.hexToUuid(event.assetInstanceID);
  const dbAssetInstance = await database.retrieveAssetInstanceByID(eventAssetInstanceID);
  if (dbAssetInstance !== null && dbAssetInstance.transactionHash !== undefined) {
    throw new Error(`Duplicate asset instance ID`);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(utils.hexToUuid(event.assetDefinitionID));
  if (assetDefinition === null) {
    throw new Error('Uknown asset definition');
  }
  if (assetDefinition.transactionHash === undefined) {
    throw new Error('Asset definition transaction must be mined');
  }
  if (assetDefinition.isContentUnique) {
    const assetInstanceByContentID = await database.retrieveAssetInstanceByDefinitionIDAndContentHash(assetDefinition.assetDefinitionID, event.contentHash);
    if (assetInstanceByContentID !== null && eventAssetInstanceID !== assetInstanceByContentID.assetInstanceID) {
      if (assetInstanceByContentID.transactionHash !== undefined) {
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
  database.upsertAssetInstance({
    assetInstanceID: eventAssetInstanceID,
    author: event.author,
    assetDefinitionID: assetDefinition.assetDefinitionID,
    descriptionHash: event.descriptionHash,
    description,
    contentHash: event.contentHash,
    content,
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  });
};

export const handleSetAssetInstancePropertyEvent = async (event: IEventAssetInstancePropertySet, blockchainData: IDBBlockchainData) => {
  const eventAssetInstanceID = utils.hexToUuid(event.assetInstanceID);
  const dbAssetInstance = await database.retrieveAssetInstanceByID(eventAssetInstanceID);
  if (dbAssetInstance === null) {
    throw new Error('Uknown asset instance');
  }
  if (dbAssetInstance.transactionHash === undefined) {
    throw new Error('Unconfirmed asset instance');
  }
  if (!event.key) {
    throw new Error('Invalid property key');
  }
  await database.setConfirmedAssetInstanceProperty(eventAssetInstanceID, event.author, event.key, event.value, Number(event.timestamp), blockchainData);
};

export const handleTradeAssetRequest = async (requesterAddress: string, assetInstanceID: string) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Uknown asset instance', 404);
  }
  if (database.isMemberOwned(assetInstance.author)) {
    throw new RequestError('Asset instance authored', 400);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetInstance.assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 500);
  }
  if (assetDefinition.contentSchema !== undefined) {
    const documentDetails = await docExchange.getDocumentDetails(utils.getUnstructuredFilePathInDocExchange(assetInstanceID));
    if (documentDetails.hash === assetInstance.contentHash) {
      throw new RequestError('Asset content already available', 400);
    }
  } else {
    if (assetInstance.content !== undefined) {
      throw new RequestError('Asset content already available', 400);
    }
  }
  const author = await database.retrieveMemberByAddress(assetInstance.author);
  if (author === null) {
    throw new RequestError('Asset author must be registered', 400);
  }
  const requester = await database.retrieveMemberByAddress(requesterAddress);
  if (requester === null) {
    throw new RequestError('Requester must be registered', 400);
  }
  const tradeRequest: IAssetTradeRequest = {
    type: 'asset-request',
    assetInstanceID,
    requester: requester.address,
    metadata: {}
  };
  app2app.dispatchMessage(requester.app2appDestination, author.app2appDestination, JSON.stringify(tradeRequest));
};
