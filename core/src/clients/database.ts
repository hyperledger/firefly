import { config } from '../lib/config';
import MongoDBProvider from './db-providers/mongodb';
import NEDBProvider from './db-providers/nedb';
import * as utils from '../lib/utils';
import { ClientEventType, IClientEventListener, IDatabaseProvider, IDBAssetDefinition, IDBAssetInstance, IDBBlockchainData, IDBMember, IDBPaymentDefinition, IDBPaymentInstance } from '../lib/interfaces';

let databaseProvider: IDatabaseProvider;

export const init = async () => {
  if (config.mongodb !== undefined) {
    databaseProvider = new MongoDBProvider();
  } else {
    databaseProvider = new NEDBProvider();
  }
  await databaseProvider.init();
};

let listeners: IClientEventListener[] = [];

// MEMBER QUERIES

export const retrieveMemberByAddress = (address: string): Promise<IDBMember | null> => {
  return databaseProvider.findOne<IDBMember>(utils.databaseCollections[utils.databaseCollectionKeys.MEMBERS].name, { address });
};

export const retrieveMembers = (query: object, skip: number, limit: number): Promise<IDBMember[]> => {
  return databaseProvider.find<IDBMember>(utils.databaseCollections[utils.databaseCollectionKeys.MEMBERS].name, query, { name: 1 }, skip, limit);
};

export const upsertMember = async (member: IDBMember) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.MEMBERS].name, { address: member.address }, { $set: member }, true);
  emitEvent('member-registered', member);
};

// ASSET DEFINITION QUERIES

export const retrieveAssetDefinitions = (query: object, skip: number, limit: number): Promise<IDBAssetDefinition[]> => {
  return databaseProvider.find<IDBAssetDefinition>(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_DEFINITIONS].name, query, { name: 1 }, skip, limit)
};

export const countAssetDefinitions = (query: object): Promise<number> => {
  return databaseProvider.count(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_DEFINITIONS].name, query);
};

export const retrieveAssetDefinitionByID = (assetDefinitionID: string): Promise<IDBAssetDefinition | null> => {
  return databaseProvider.findOne<IDBAssetDefinition>(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_DEFINITIONS].name, { assetDefinitionID });
};

export const retrieveAssetDefinitionByName = (name: string): Promise<IDBAssetDefinition | null> => {
  return databaseProvider.findOne<IDBAssetDefinition>(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_DEFINITIONS].name, { name });
};

export const upsertAssetDefinition = async (assetDefinition: IDBAssetDefinition) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_DEFINITIONS].name, { assetDefinitionID: assetDefinition.assetDefinitionID }, { $set: assetDefinition }, true);
  if (assetDefinition.submitted !== undefined) {
    emitEvent('asset-definition-submitted', assetDefinition);
  } else if (assetDefinition.transactionHash !== undefined) {
    emitEvent('asset-definition-created', assetDefinition);
  }
};

export const markAssetDefinitionAsConflict = async (assetDefinitionID: string, timestamp: number) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_DEFINITIONS].name, { assetDefinitionID }, { $set: { timestamp, conflict: true } }, false);
  emitEvent('asset-definition-name-conflict', { assetDefinitionID })
};

// PAYMENT DEFINITION QUERIES

export const retrievePaymentDefinitions = (query: object, skip: number, limit: number): Promise<IDBPaymentDefinition[]> => {
  return databaseProvider.find<IDBPaymentDefinition>(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_DEFINITIONS].name, query, { name: 1 }, skip, limit);
};

export const countPaymentDefinitions = (query: object): Promise<number> => {
  return databaseProvider.count(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_DEFINITIONS].name, query);
};

export const retrievePaymentDefinitionByID = (paymentDefinitionID: string): Promise<IDBPaymentDefinition | null> => {
  return databaseProvider.findOne<IDBPaymentDefinition>(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_DEFINITIONS].name, { paymentDefinitionID });
};

export const retrievePaymentDefinitionByName = (name: string): Promise<IDBPaymentDefinition | null> => {
  return databaseProvider.findOne<IDBPaymentDefinition>(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_DEFINITIONS].name, { name });
};

export const upsertPaymentDefinition = async (paymentDefinition: IDBPaymentDefinition) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_DEFINITIONS].name, { paymentDefinitionID: paymentDefinition.paymentDefinitionID }, { $set: paymentDefinition }, true)
  if (paymentDefinition.submitted !== undefined) {
    emitEvent('payment-definition-submitted', paymentDefinition);
  } else if (paymentDefinition.transactionHash !== undefined) {
    emitEvent('payment-definition-created', paymentDefinition);
  }
};

export const markPaymentDefinitionAsConflict = async (paymentDefinitionID: string, timestamp: number) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_DEFINITIONS].name, { paymentDefinitionID }, { $set: { conflict: true, timestamp } }, false);
  emitEvent('payment-definition-name-conflict', { paymentDefinitionID })
};

// ASSET INSTANCE QUERIES

export const retrieveAssetInstances = (query: object, skip: number, limit: number): Promise<IDBAssetInstance[]> => {
  return databaseProvider.find<IDBAssetInstance>(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, query, {}, skip, limit);
};

export const countAssetInstances = (query: object): Promise<number> => {
  return databaseProvider.count(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, query);
};

export const retrieveAssetInstanceByID = (assetInstanceID: string): Promise<IDBAssetInstance | null> => {
  return databaseProvider.findOne<IDBAssetInstance>(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetInstanceID });
};

export const retrieveAssetInstanceByDefinitionIDAndContentHash = (assetDefinitionID: string, contentHash: string): Promise<IDBAssetInstance | null> => {
  return databaseProvider.findOne<IDBAssetInstance>(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetDefinitionID, contentHash });
};

export const upsertAssetInstance = async (assetInstance: IDBAssetInstance) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetInstanceID: assetInstance.assetInstanceID }, { $set: assetInstance }, true);
  if (assetInstance.submitted !== undefined) {
    emitEvent('asset-instance-submitted', assetInstance);
  } else if (assetInstance.transactionHash !== undefined) {
    emitEvent('asset-instance-created', assetInstance);
  }
};

export const setAssetInstancePrivateContent = async (assetInstanceID: string, content: object | undefined, filename: string | undefined) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetInstanceID }, { $set: { content, filename } }, true);
  emitEvent('private-asset-instance-content-stored', { assetInstanceID, content, filename });
};

export const markAssetInstanceAsConflict = async (assetInstanceID: string, timestamp: number) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetInstanceID }, { $set: { conflict: true, timestamp } }, false);
  emitEvent('asset-instance-content-conflict', { assetInstanceID });
};

export const setSubmittedAssetInstanceProperty = async (assetInstanceID: string, author: string, key: string, value: string, submitted: number, receipt: string | undefined) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetInstanceID },
    {
      $set: {
        [`properties.${author}.${key}.value`]: value,
        [`properties.${author}.${key}.submitted`]: submitted,
        [`properties.${author}.${key}.receipt`]: receipt
      }
    }, false);
  emitEvent('asset-instance-property-submitted', { assetInstanceID, key, value, submitted, receipt });
};

export const setConfirmedAssetInstanceProperty = async (assetInstanceID: string, author: string, key: string, value: string, timestamp: number, { blockNumber, transactionHash }: IDBBlockchainData) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.ASSET_INSTANCES].name, { assetInstanceID },
    {
      $set: {
        [`properties.${author}.${key}.value`]: value,
        [`properties.${author}.${key}.history.${timestamp}`]: { value, timestamp, blockNumber, transactionHash }
      }
    }, false);
  emitEvent('asset-instance-property-set', { assetInstanceID, key, value, timestamp, blockNumber, transactionHash });
};

// PAYMENT INSTANCE QUERIES

export const retrievePaymentInstances = (query: object, skip: number, limit: number): Promise<IDBPaymentInstance[]> => {
  return databaseProvider.find<IDBPaymentInstance>(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_INSTANCES].name, query, {}, skip, limit);
};

export const countPaymentInstances = (query: object): Promise<number> => {
  return databaseProvider.count(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_INSTANCES].name, query);
};

export const retrievePaymentInstanceByID = (paymentInstanceID: string): Promise<IDBPaymentInstance | null> => {
  return databaseProvider.findOne<IDBPaymentInstance>(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_INSTANCES].name, { paymentInstanceID });
};

export const upsertPaymentInstance = async (paymentInstance: IDBPaymentInstance) => {
  await databaseProvider.updateOne(utils.databaseCollections[utils.databaseCollectionKeys.PAYMENT_INSTANCES].name, { paymentInstanceID: paymentInstance.paymentInstanceID }, { $set: paymentInstance }, true);
  if (paymentInstance.submitted !== undefined) {
    emitEvent('payment-instance-submitted', paymentInstance);
  } else {
    emitEvent('payment-instance-created', paymentInstance);
  }
};

// EVENT HANDLING

export const addListener = (listener: IClientEventListener) => {
  listeners.push(listener);
};

export const removeListener = (listener: IClientEventListener) => {
  listeners = listeners.filter(entry => entry != listener);
};

const emitEvent = (eventType: ClientEventType, content: object) => {
  for (const listener of listeners) {
    listener(eventType, content);
  }
};

export const shutDown = () => {
  databaseProvider.shutDown();
};