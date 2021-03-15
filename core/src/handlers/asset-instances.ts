import Ajv from 'ajv';
import { createLogger, LogLevelString } from 'bunyan';
import { v4 as uuidV4 } from 'uuid';
import * as apiGateway from '../clients/api-gateway';
import * as app2app from '../clients/app2app';
import * as database from '../clients/database';
import * as docExchange from '../clients/doc-exchange';
import * as ipfs from '../clients/ipfs';
import { config } from '../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IAssetInstance, IAssetTradePrivateAssetInstancePush, IDBAssetInstance, IDBBlockchainData, IEventAssetInstanceBatchCreated, IEventAssetInstanceCreated, IEventAssetInstancePropertySet, IPendingAssetInstancePrivateContentDelivery } from '../lib/interfaces';
import RequestError from '../lib/request-error';
import * as utils from '../lib/utils';
import { assetInstancesPinning } from './asset-instances-pinning';
import * as assetTrade from './asset-trade';

const log = createLogger({ name: 'handlers/asset-instances.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

const ajv = new Ajv();

export let pendingAssetInstancePrivateContentDeliveries: { [assetInstanceID: string]: IPendingAssetInstancePrivateContentDelivery } = {};

export const handleGetAssetInstancesRequest = (assetDefinitionID: string, query: object, sort: object, skip: number, limit: number) => {
  return database.retrieveAssetInstances(assetDefinitionID, query, sort, skip, limit);
};

export const handleCountAssetInstancesRequest = async (assetDefinitionID: string, query: object) => {
  return { count: await database.countAssetInstances(assetDefinitionID, query) };
};

export const handleGetAssetInstanceRequest = async (assetDefinitionID: string, assetInstanceID: string, content: boolean) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetDefinitionID, assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Asset instance not found', 404);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Asset definition not found', 500);
  }
  if (content) {
    if (assetDefinition.contentSchema) {
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

export const handleCreateStructuredAssetInstanceRequest = async (author: string, assetDefinitionID: string, description: Object | undefined, content: Object, isContentPrivate: boolean | undefined, participants: string[] | undefined, sync: boolean) => {
  let descriptionHash: string | undefined;
  let contentHash: string;
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 400);
  }
  if (assetDefinition.conflict === true) {
    throw new RequestError('Cannot instantiate assets of conflicted definition', 400);
  }
  // For ethereum, we need to make assert definition transaction is mined
  if (config.protocol === 'ethereum' && assetDefinition.transactionHash === undefined) {
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
    descriptionHash = `0x${utils.getSha256(JSON.stringify(description))}`;
  }
  if (!ajv.validate(assetDefinition.contentSchema, content)) {
    throw new RequestError('Content does not conform to asset definition schema', 400);
  }
  if(isContentPrivate === undefined) {
    isContentPrivate = assetDefinition.isContentPrivate;
  }
  contentHash = `0x${utils.getSha256(JSON.stringify(content))}`;
  if (assetDefinition.isContentUnique && (await database.retrieveAssetInstanceByDefinitionIDAndContentHash(assetDefinition.assetDefinitionID, contentHash)) !== null) {
    throw new RequestError(`Asset instance content conflict`);
  }
  if (config.protocol === 'corda') {
    // validate participants are registered members
    if (participants !== undefined) {
      for (const participant of participants) {
        if (await database.retrieveMemberByAddress(participant) === null) {
          throw new RequestError('One or more participants are not registered', 400);
        }
      }
    } else {
      throw new RequestError('Missing asset participants', 400);
    }
  }
  const assetInstanceID = uuidV4();
  const timestamp = utils.getTimestamp();
  const assetInstance: IAssetInstance = {
    assetInstanceID,
    author,
    assetDefinitionID,
    descriptionHash,
    description,
    contentHash,
    content,
    isContentPrivate
  };

  let dbAssetInstance: IDBAssetInstance = assetInstance;
  dbAssetInstance.submitted = timestamp;
  if (config.protocol === 'corda') {
    dbAssetInstance.participants = participants;
  }
  // If there are public IPFS shared parts of this instance, we can batch it together with all other
  // assets we are publishing for performance. Reducing both the data we write to the blockchain, and
  // most importantly the number of IPFS transactions.
  // Curently we do batching only for ethereum
  if ((assetDefinition.descriptionSchema || !isContentPrivate) && config.protocol === 'ethereum') {
    dbAssetInstance.batchID = await assetInstancesPinning.pin(assetInstance);
    await database.upsertAssetInstance(dbAssetInstance);
  } else {
    await database.upsertAssetInstance(dbAssetInstance);
    // One-for-one blockchain transactions to instances
    let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
    if (descriptionHash) {
      apiGatewayResponse = await apiGateway.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, participants, sync);
    } else {
      apiGatewayResponse = await apiGateway.createAssetInstance(assetInstanceID, assetDefinitionID, author, contentHash, participants, sync);
    }
    // dbAssetInstance.receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
    if(apiGatewayResponse.type === 'async') {
      await database.setAssetInstanceReceipt(assetDefinitionID, assetInstanceID, apiGatewayResponse.id);
    }
  }
  return assetInstanceID;
};

export const handleCreateUnstructuredAssetInstanceRequest = async (author: string, assetDefinitionID: string, description: Object | undefined, content: NodeJS.ReadableStream, filename: string, isContentPrivate: boolean | undefined, participants: string[] | undefined, sync: boolean) => {
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
    descriptionHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(description)));
  }
  if(isContentPrivate === undefined) {
    isContentPrivate = assetDefinition.isContentPrivate;
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
  if (config.protocol === 'corda') {
    // validate participants are registered
    if (participants) {
      for (const participant of participants) {
        if (await database.retrieveMemberByAddress(participant) === null) {
          throw new RequestError(`One or more participants are not registered`, 400);
        }
      }
    } else {
      throw new RequestError(`Missing asset participants`, 400);
    }
  }
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
  const timestamp = utils.getTimestamp();
  await database.upsertAssetInstance({
    assetInstanceID,
    author,
    assetDefinitionID,
    descriptionHash,
    description,
    contentHash,
    filename,
    isContentPrivate,
    participants,
    submitted: timestamp
  });
  if (descriptionHash) {
    apiGatewayResponse = await apiGateway.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, participants, sync);
  } else {
    apiGatewayResponse = await apiGateway.createAssetInstance(assetInstanceID, assetDefinitionID, author, contentHash, participants, sync);
  }
  if(apiGatewayResponse.type === 'async') {
    await database.setAssetInstanceReceipt(assetDefinitionID, assetInstanceID, apiGatewayResponse.id);
  }
  return assetInstanceID;
}

export const handleSetAssetInstancePropertyRequest = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string, sync: boolean) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetDefinitionID, assetInstanceID);
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
  const submitted = utils.getTimestamp();
  await database.setSubmittedAssetInstanceProperty(assetDefinitionID, assetInstanceID, author, key, value, submitted);
  const apiGatewayResponse = await apiGateway.setAssetInstanceProperty(assetDefinitionID, assetInstanceID, author, key, value, assetInstance.participants, sync);
  if(apiGatewayResponse.type === 'async') {
    await database.setAssetInstancePropertyReceipt(assetDefinitionID, assetInstanceID, author, key, apiGatewayResponse.id);
  }
};

export const handleAssetInstanceBatchCreatedEvent = async (event: IEventAssetInstanceBatchCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {

  let batch = await database.retrieveBatchByHash(event.batchHash);
  if (!batch) {
    batch = await ipfs.downloadJSON(utils.sha256ToIPFSHash(event.batchHash));
  }
  if (!batch) {
    throw new Error('Unknown batch hash: ' + event.batchHash);
  }

  // Process each record within the batch, as if it is an individual event
  const records: IAssetInstance[] = batch.records || [];
  for (let record of records) {
    const recordEvent: IEventAssetInstanceCreated = {
      assetDefinitionID: '',
      assetInstanceID: '',
      author: record.author,
      contentHash: record.contentHash!,
      descriptionHash: record.descriptionHash!,
      timestamp: event.timestamp,
      isContentPrivate: record.isContentPrivate
    };
    try {
      await handleAssetInstanceCreatedEvent(recordEvent, { blockNumber, transactionHash }, record);
    } catch (err) {
      // We failed to process this record, but continue to attempt the other records in the batch
      log.error(`Record ${record.assetDefinitionID}/${record.assetInstanceID} in batch ${batch.batchID} with hash ${event.batchHash} failed`, err.stack);
    }
  }

  // Process each property within the batch, as if it is an individual event
  // Note we process these after the records, to ensure asset creation always comes after setting properties
  const properties: IEventAssetInstancePropertySet[] = batch.properties || [];
  for (let property of properties) {
    try {
      await handleSetAssetInstancePropertyEvent(property, { blockNumber, transactionHash });
    } catch (err) {
      // We failed to process this record, but continue to attempt the other records in the batch
      log.error(`Property ${property.assetDefinitionID}/${property.assetInstanceID}/${property.key} in batch ${batch.batchID} with hash ${event.batchHash} failed`, err.stack);
    }
  }

  // Write the batch itself to our local database
  await database.upsertBatch({
    ...batch,
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  });

}

export const handleAssetInstanceCreatedEvent = async (event: IEventAssetInstanceCreated, { blockNumber, transactionHash }: IDBBlockchainData, batchInstance?: IAssetInstance) => {
  let eventAssetInstanceID: string;
  let eventAssetDefinitionID: string;
  if (batchInstance === undefined) {
    switch (config.protocol) {
      case 'corda':
        eventAssetInstanceID = event.assetInstanceID;
        eventAssetDefinitionID = event.assetDefinitionID;
        break;
      case 'ethereum':
        eventAssetInstanceID = utils.hexToUuid(event.assetInstanceID);
        eventAssetDefinitionID = utils.hexToUuid(event.assetDefinitionID);
        break;
    }
  } else {
    eventAssetInstanceID = batchInstance.assetInstanceID;
    eventAssetDefinitionID = batchInstance.assetDefinitionID;
    log.info(`batch instance ${eventAssetDefinitionID}:${eventAssetInstanceID}`);
  }
  const dbAssetInstance = await database.retrieveAssetInstanceByID(eventAssetDefinitionID, eventAssetInstanceID);
  if (dbAssetInstance !== null && dbAssetInstance.transactionHash !== undefined) {
    throw new Error(`Duplicate asset instance ID`);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(eventAssetDefinitionID);
  if (assetDefinition === null) {
    throw new Error('Unkown asset definition');
  }
  // For ethereum, we need to make asset definition transaction is mined
  if (config.protocol === 'ethereum' && assetDefinition.transactionHash === undefined) {
    throw new Error('Asset definition transaction must be mined');
  }
  if (assetDefinition.isContentUnique) {
    const assetInstanceByContentID = await database.retrieveAssetInstanceByDefinitionIDAndContentHash(eventAssetDefinitionID, event.contentHash);
    if (assetInstanceByContentID !== null && eventAssetInstanceID !== assetInstanceByContentID.assetInstanceID) {
      if (assetInstanceByContentID.transactionHash !== undefined) {
        throw new Error(`Asset instance content conflict ${event.contentHash}`);
      } else {
        await database.markAssetInstanceAsConflict(eventAssetDefinitionID, assetInstanceByContentID.assetInstanceID, Number(event.timestamp));
      }
    }
  }
  let description: Object | undefined = batchInstance?.description;
  if (assetDefinition.descriptionSchema && !description) {
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
  let content: Object | undefined = batchInstance?.content;
  if (assetDefinition.contentSchema && !content) {
    if (event.contentHash === dbAssetInstance?.contentHash) {
      content = dbAssetInstance.content;
    } else if (!assetDefinition.isContentPrivate) {
      content = await ipfs.downloadJSON(utils.sha256ToIPFSHash(event.contentHash));
      if (!ajv.validate(assetDefinition.contentSchema, content)) {
        throw new Error('Content does not conform to schema');
      }
    }
  }
  log.trace(`Updating asset instance ${eventAssetInstanceID} with blockchain pinned info blockNumber=${blockNumber} hash=${transactionHash}`);
  let assetInstanceDB: IDBAssetInstance = {
    assetInstanceID: eventAssetInstanceID,
    author: event.author,
    assetDefinitionID: assetDefinition.assetDefinitionID,
    descriptionHash: event.descriptionHash,
    description,
    contentHash: event.contentHash,
    timestamp: Number(event.timestamp),
    content,
    blockNumber,
    transactionHash,
    isContentPrivate: event.isContentPrivate
  };
  if (config.protocol === 'corda') {
    assetInstanceDB.participants = event.participants;
  }
  await database.upsertAssetInstance(assetInstanceDB);
  if (assetInstanceDB.isContentPrivate) {
    const privateData = pendingAssetInstancePrivateContentDeliveries[eventAssetInstanceID];
    if (privateData !== undefined) {
      const author = await database.retrieveMemberByAddress(event.author);
      if (author === null) {
        throw new Error('Pending private data author unknown');
      }
      if (author.app2appDestination !== privateData.fromDestination) {
        throw new Error('Pending private data destination mismatch');
      }
      if (privateData.content !== undefined) {
        const privateDataHash = `0x${utils.getSha256(JSON.stringify(privateData.content))}`;
        if (privateDataHash !== event.contentHash) {
          throw new Error('Pending private data content hash mismatch');
        }
      }
      await database.setAssetInstancePrivateContent(eventAssetDefinitionID, eventAssetInstanceID, privateData.content, privateData.filename);
      delete pendingAssetInstancePrivateContentDeliveries[eventAssetInstanceID];
    }
  }
};

export const handleSetAssetInstancePropertyEvent = async (event: IEventAssetInstancePropertySet, blockchainData: IDBBlockchainData) => {
  let eventAssetInstanceID: string;
  let eventAssetDefinitionID: string;
  switch (config.protocol) {
    case 'corda':
      eventAssetInstanceID = event.assetInstanceID;
      eventAssetDefinitionID = event.assetDefinitionID;
      break;
    case 'ethereum':
      eventAssetInstanceID = utils.hexToUuid(event.assetInstanceID);
      eventAssetDefinitionID = utils.hexToUuid(event.assetDefinitionID);
      break;
  }
  const dbAssetInstance = await database.retrieveAssetInstanceByID(eventAssetDefinitionID, eventAssetInstanceID);
  if (dbAssetInstance === null) {
    throw new Error('Uknown asset instance');
  }
  if (dbAssetInstance.transactionHash === undefined) {
    throw new Error('Unconfirmed asset instance');
  }
  if (!event.key) {
    throw new Error('Invalid property key');
  }
  await database.setConfirmedAssetInstanceProperty(eventAssetDefinitionID, eventAssetInstanceID, event.author, event.key, event.value, Number(event.timestamp), blockchainData);
};

export const handleAssetInstanceTradeRequest = async (assetDefinitionID: string, requesterAddress: string, assetInstanceID: string, metadata: object | undefined) => {
  const assetInstance = await database.retrieveAssetInstanceByID(assetDefinitionID, assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Uknown asset instance', 404);
  }
  const author = await database.retrieveMemberByAddress(assetInstance.author);
  if (author === null) {
    throw new RequestError('Asset author must be registered', 400);
  }
  if (author.assetTrailInstanceID === config.assetTrailInstanceID) {
    throw new RequestError('Asset instance authored', 400);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 500);
  }
  if (assetDefinition.contentSchema !== undefined) {
    if (assetInstance.content !== undefined) {
      throw new RequestError('Asset content already available', 400);
    }
  } else {
    try {
      const documentDetails = await docExchange.getDocumentDetails(utils.getUnstructuredFilePathInDocExchange(assetInstanceID));
      if (documentDetails.hash === assetInstance.contentHash) {
        throw new RequestError('Asset content already available', 400);
      }
    } catch (err) {
      if (err.response?.status !== 404) {
        throw new RequestError(err, 500);
      }
    }
  }
  const requester = await database.retrieveMemberByAddress(requesterAddress);
  if (requester === null) {
    throw new RequestError('Requester must be registered', 400);
  }
  await assetTrade.coordinateAssetTrade(assetInstance, assetDefinition, requester.address, metadata, author.app2appDestination);
};

export const handlePushPrivateAssetInstanceRequest = async (assetDefinitionID: string, assetInstanceID: string, recipientAddress: string) => {
  const recipient = await database.retrieveMemberByAddress(recipientAddress);
  if (recipient === null) {
    throw new RequestError('Unknown recipient', 400);
  }
  const assetInstance = await database.retrieveAssetInstanceByID(assetDefinitionID, assetInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Unknown asset instance', 400);
  }
  const author = await database.retrieveMemberByAddress(assetInstance.author);
  if (author === null) {
    throw new RequestError('Unknown asset author', 500);
  }
  if (author.assetTrailInstanceID !== config.assetTrailInstanceID) {
    throw new RequestError('Must be asset instance author', 403);
  }
  const assetDefinition = await database.retrieveAssetDefinitionByID(assetInstance.assetDefinitionID);
  if (assetDefinition === null) {
    throw new RequestError('Unknown asset definition', 500);
  }
  let privateAssetTradePrivateInstancePush: IAssetTradePrivateAssetInstancePush = {
    type: 'private-asset-instance-push',
    assetInstanceID,
    assetDefinitionID
  };
  if (assetDefinition.contentSchema !== undefined) {
    privateAssetTradePrivateInstancePush.content = assetInstance.content;
  } else {
    await docExchange.transfer(author.docExchangeDestination, recipient.docExchangeDestination,
      utils.getUnstructuredFilePathInDocExchange(assetInstanceID));
    privateAssetTradePrivateInstancePush.filename = assetInstance.filename;
  }
  app2app.dispatchMessage(recipient.app2appDestination, privateAssetTradePrivateInstancePush);
};
