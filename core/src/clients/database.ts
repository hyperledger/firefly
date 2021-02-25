import { config } from '../lib/config';
import MongoDBProvider from './db-providers/mongodb';
import NEDBProvider from './db-providers/nedb';
import { ClientEventType, IClientEventListener, IDatabaseProvider, IDBAssetDefinition, IDBAssetInstance, IDBBatch, IDBBlockchainData, IDBMember, IDBPaymentDefinition, IDBPaymentInstance } from '../lib/interfaces';

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

// COLLECTION AGNOSTIC QUERIES

export const createCollection = (collectionName: string, indexes: { fields: string[], unique?: boolean }[]) => {
  return databaseProvider.createCollection(collectionName, indexes);
};

// MEMBER QUERIES

export const retrieveMemberByAddress = (address: string): Promise<IDBMember | null> => {
  return databaseProvider.findOne<IDBMember>('members', { address });
};

export const retrieveMembers = (query: object, skip: number, limit: number): Promise<IDBMember[]> => {
  return databaseProvider.find<IDBMember>('members', query, { name: 1 }, skip, limit);
};

export const upsertMember = async (member: IDBMember) => {
  await databaseProvider.updateOne('members', { address: member.address }, { $set: member }, true);
  emitEvent('member-registered', member);
};

// ASSET DEFINITION QUERIES

export const retrieveAssetDefinitions = (query: object, skip: number, limit: number): Promise<IDBAssetDefinition[]> => {
  return databaseProvider.find<IDBAssetDefinition>('asset-definitions', query, { name: 1 }, skip, limit)
};

export const countAssetDefinitions = (query: object): Promise<number> => {
  return databaseProvider.count('asset-definitions', query);
};

export const retrieveAssetDefinitionByID = (assetDefinitionID: string): Promise<IDBAssetDefinition | null> => {
  return databaseProvider.findOne<IDBAssetDefinition>('asset-definitions', { assetDefinitionID });
};

export const retrieveAssetDefinitionByName = (name: string): Promise<IDBAssetDefinition | null> => {
  return databaseProvider.findOne<IDBAssetDefinition>('asset-definitions', { name });
};

export const upsertAssetDefinition = async (assetDefinition: IDBAssetDefinition) => {
  await databaseProvider.updateOne('asset-definitions', { assetDefinitionID: assetDefinition.assetDefinitionID }, { $set: assetDefinition }, true);
  if (assetDefinition.submitted !== undefined) {
    emitEvent('asset-definition-submitted', assetDefinition);
  } else if (assetDefinition.transactionHash !== undefined) {
    emitEvent('asset-definition-created', assetDefinition);
  }
};

export const markAssetDefinitionAsConflict = async (assetDefinitionID: string, timestamp: number) => {
  await databaseProvider.updateOne('asset-definitions', { assetDefinitionID }, { $set: { timestamp, conflict: true } }, false);
  emitEvent('asset-definition-name-conflict', { assetDefinitionID })
};

// PAYMENT DEFINITION QUERIES

export const retrievePaymentDefinitions = (query: object, skip: number, limit: number): Promise<IDBPaymentDefinition[]> => {
  return databaseProvider.find<IDBPaymentDefinition>('payment-definitions', query, { name: 1 }, skip, limit);
};

export const countPaymentDefinitions = (query: object): Promise<number> => {
  return databaseProvider.count('payment-definitions', query);
};

export const retrievePaymentDefinitionByID = (paymentDefinitionID: string): Promise<IDBPaymentDefinition | null> => {
  return databaseProvider.findOne<IDBPaymentDefinition>('payment-definitions', { paymentDefinitionID });
};

export const retrievePaymentDefinitionByName = (name: string): Promise<IDBPaymentDefinition | null> => {
  return databaseProvider.findOne<IDBPaymentDefinition>('payment-definitions', { name });
};

export const upsertPaymentDefinition = async (paymentDefinition: IDBPaymentDefinition) => {
  await databaseProvider.updateOne('payment-definitions', { paymentDefinitionID: paymentDefinition.paymentDefinitionID }, { $set: paymentDefinition }, true)
  if (paymentDefinition.submitted !== undefined) {
    emitEvent('payment-definition-submitted', paymentDefinition);
  } else if (paymentDefinition.transactionHash !== undefined) {
    emitEvent('payment-definition-created', paymentDefinition);
  }
};

export const markPaymentDefinitionAsConflict = async (paymentDefinitionID: string, timestamp: number) => {
  await databaseProvider.updateOne('payment-definitions', { paymentDefinitionID }, { $set: { conflict: true, timestamp } }, false);
  emitEvent('payment-definition-name-conflict', { paymentDefinitionID })
};

// ASSET INSTANCE QUERIES

export const retrieveAssetInstances = (assetDefinitionID: string, query: object, sort: object, skip: number, limit: number): Promise<IDBAssetInstance[]> => {
  return databaseProvider.find<IDBAssetInstance>(`asset-instance-${assetDefinitionID}`, query, sort, skip, limit);
};

export const countAssetInstances = (assetDefinitionID: string, query: object): Promise<number> => {
  return databaseProvider.count(`asset-instance-${assetDefinitionID}`, query);
};

export const retrieveAssetInstanceByID = (assetDefinitionID: string, assetInstanceID: string): Promise<IDBAssetInstance | null> => {
  return databaseProvider.findOne<IDBAssetInstance>(`asset-instance-${assetDefinitionID}`, { assetInstanceID });
};

export const retrieveAssetInstanceByDefinitionIDAndContentHash = (assetDefinitionID: string, contentHash: string): Promise<IDBAssetInstance | null> => {
  return databaseProvider.findOne<IDBAssetInstance>(`asset-instance-${assetDefinitionID}`, { contentHash });
};

export const upsertAssetInstance = async (assetInstance: IDBAssetInstance) => {
  await databaseProvider.updateOne(`asset-instance-${assetInstance.assetDefinitionID}`, { assetInstanceID: assetInstance.assetInstanceID }, { $set: assetInstance }, true);
  if (assetInstance.submitted !== undefined) {
    emitEvent('asset-instance-submitted', assetInstance);
  } else if (assetInstance.transactionHash !== undefined) {
    emitEvent('asset-instance-created', assetInstance);
  }
};

export const setAssetInstancePrivateContent = async (assetDefinitionID: string, assetInstanceID: string, content: object | undefined, filename: string | undefined) => {
  await databaseProvider.updateOne(`asset-instance-${assetDefinitionID}`, { assetInstanceID }, { $set: { content, filename } }, true);
  emitEvent('private-asset-instance-content-stored', { assetDefinitionID, assetInstanceID, content, filename });
};

export const markAssetInstanceAsConflict = async (assetDefinitionID: string, assetInstanceID: string, timestamp: number) => {
  await databaseProvider.updateOne(`asset-instance-${assetDefinitionID}`, { assetInstanceID }, { $set: { conflict: true, timestamp } }, false);
  emitEvent('asset-instance-content-conflict', { assetInstanceID });
};

export const setSubmittedAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string, submitted: number, receipt: string | undefined) => {
  await databaseProvider.updateOne(`asset-instance-${assetDefinitionID}`, { assetInstanceID },
    {
      $set: {
        [`properties.${author}.${key}.value`]: value,
        [`properties.${author}.${key}.submitted`]: submitted,
        [`properties.${author}.${key}.receipt`]: receipt
      }
    }, false);
  emitEvent('asset-instance-property-submitted', { assetInstanceID, key, value, submitted, receipt });
};

export const setConfirmedAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string, timestamp: number, { blockNumber, transactionHash }: IDBBlockchainData) => {
  await databaseProvider.updateOne(`asset-instance-${assetDefinitionID}`, { assetInstanceID },
    {
      $set: {
        [`properties.${author}.${key}.value`]: value,
        [`properties.${author}.${key}.history.${timestamp}`]: { value, timestamp, blockNumber, transactionHash }
      }
    }, false);
  emitEvent('asset-instance-property-set', { assetInstanceID, author, key, value, timestamp, blockNumber, transactionHash });
};

// PAYMENT INSTANCE QUERIES

export const retrievePaymentInstances = (query: object, skip: number, limit: number): Promise<IDBPaymentInstance[]> => {
  return databaseProvider.find<IDBPaymentInstance>('payment-instances', query, {}, skip, limit);
};

export const countPaymentInstances = (query: object): Promise<number> => {
  return databaseProvider.count('payment-instances', query);
};

export const retrievePaymentInstanceByID = (paymentInstanceID: string): Promise<IDBPaymentInstance | null> => {
  return databaseProvider.findOne<IDBPaymentInstance>('payment-instances', { paymentInstanceID });
};

export const upsertPaymentInstance = async (paymentInstance: IDBPaymentInstance) => {
  await databaseProvider.updateOne('payment-instances', { paymentInstanceID: paymentInstance.paymentInstanceID }, { $set: paymentInstance }, true);
  if (paymentInstance.submitted !== undefined) {
    emitEvent('payment-instance-submitted', paymentInstance);
  } else {
    emitEvent('payment-instance-created', paymentInstance);
  }
};


// BATCH QUERIES

export const retrieveBatches = (query: object, skip: number, limit: number, sort: {[f: string]: number} = {}): Promise<IDBBatch<any>[]> => {
  return databaseProvider.find<IDBBatch<any>>('batches', query, sort, skip, limit);
};

export const retrieveBatchByID = (batchID: string): Promise<IDBBatch<any> | null> => {
  return databaseProvider.findOne<IDBBatch<any>>('batches', { batchID });
};

export const retrieveBatchByHash = (batchHash: string): Promise<IDBBatch<any> | null> => {
  return databaseProvider.findOne<IDBBatch<any>>('batches', { batchHash });
};

export const upsertBatch = async (batch: IDBBatch<any>) => {
  await databaseProvider.updateOne('batches', { batchID: batch.batchID }, { $set: batch }, true);
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
