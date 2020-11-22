import Datastore from 'nedb-promises';
import { constants } from '../lib/utils';
import path from 'path';
import { ClientEventType, IClientEventListener, IDBAssetDefinition, IDBAssetInstance, IDBBlockchainData, IDBMember, IDBPaymentDefinition, IDBPaymentInstance } from '../lib/interfaces';

let listeners: IClientEventListener[] = [];

const membersDb = Datastore.create({
  filename: path.join(constants.DATA_DIRECTORY, constants.MEMBERS_DATABASE_FILE_NAME),
  autoload: true
});

const assetDefinitionsDb = Datastore.create({
  filename: path.join(constants.DATA_DIRECTORY, constants.ASSET_DEFINITIONS_DATABASE_FILE_NAME),
  autoload: true
});

const paymentDefinitionsDb = Datastore.create({
  filename: path.join(constants.DATA_DIRECTORY, constants.PAYMENT_DEFINITIONS_DATABASE_FILE_NAME),
  autoload: true
});

const assetInstancesDb = Datastore.create({
  filename: path.join(constants.DATA_DIRECTORY, constants.ASSET_INSTANCES_DATABASE_FILE_NAME),
  autoload: true
});

const paymentInstancesDb = Datastore.create({
  filename: path.join(constants.DATA_DIRECTORY, constants.PAYMENT_INSTANCES_DATABASE_FILE_NAME),
  autoload: true
});

membersDb.ensureIndex({ fieldName: 'address', unique: true });
assetDefinitionsDb.ensureIndex({ fieldName: 'assetDefinitionID', unique: true });
paymentDefinitionsDb.ensureIndex({ fieldName: 'paymentDefinitionID', unique: true });
assetInstancesDb.ensureIndex({ fieldName: 'assetInstanceID', unique: true });
paymentInstancesDb.ensureIndex({ fieldName: 'paymentInstanceID', unique: true });

// MEMBER QUERIES

export const retrieveMemberByAddress = (address: string): Promise<IDBMember | null> => {
  return membersDb.findOne<IDBMember>({ address }, { _id: 0 });
};

export const retrieveMembers = (query: object, skip: number, limit: number): Promise<IDBMember[]> => {
  return membersDb.find<IDBMember>(query, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 });
};

export const upsertMember = (member: IDBMember) => {
  return membersDb.update({ address: member.address }, {
    $set: member
  }, { upsert: true });
};

// ASSET DEFINITION QUERIES

export const retrieveAssetDefinitions = (query: object, skip: number, limit: number): Promise<IDBAssetDefinition[]> => {
  return assetDefinitionsDb.find<IDBAssetDefinition>(query, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 });
};

export const countAssetDefinitions = (query: object): Promise<number> => {
  return assetDefinitionsDb.count(query);
};

export const retrieveAssetDefinitionByID = (assetDefinitionID: string): Promise<IDBAssetDefinition | null> => {
  return assetDefinitionsDb.findOne<IDBAssetDefinition>({ assetDefinitionID }, { _id: 0 });
};

export const retrieveAssetDefinitionByName = (name: string): Promise<IDBAssetDefinition | null> => {
  return assetDefinitionsDb.findOne<IDBAssetDefinition>({ name }, { _id: 0 });
};

export const upsertAssetDefinition = async (assetDefinition: IDBAssetDefinition) => {
  await assetDefinitionsDb.update({ assetDefinitionID: assetDefinition.assetDefinitionID }, {
    $set: assetDefinition
  }, { upsert: true });
  if(assetDefinition.submitted !== undefined) {
    emitEvent('asset-definition-submitted', assetDefinition);
  } else if(assetDefinition.transactionHash !== undefined) {
    emitEvent('asset-definition-created', assetDefinition);
  }
};

export const markAssetDefinitionAsConflict = async (assetDefinitionID: string, timestamp: number) => {
  await assetDefinitionsDb.update({ assetDefinitionID }, { $set: { timestamp, conflict: true } });
  emitEvent('asset-definition-name-conflict', { assetDefinitionID })
};

// PAYMENT DEFINITION QUERIES

export const retrievePaymentDefinitions = (query: object, skip: number, limit: number): Promise<IDBPaymentDefinition[]> => {
  return paymentDefinitionsDb.find<IDBPaymentDefinition>(query, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 })
};

export const countPaymentDefinitions = (query: object,): Promise<number> => {
  return paymentDefinitionsDb.count(query);
};

export const retrievePaymentDefinitionByID = (paymentDefinitionID: string): Promise<IDBPaymentDefinition | null> => {
  return paymentDefinitionsDb.findOne<IDBPaymentDefinition>({ paymentDefinitionID }, { _id: 0 });
};

export const retrievePaymentDefinitionByName = (name: string): Promise<IDBPaymentDefinition | null> => {
  return paymentDefinitionsDb.findOne<IDBPaymentDefinition>({ name }, { _id: 0 });
};

export const upsertPaymentDefinition = async (paymentDefinition: IDBPaymentDefinition) => {
  await paymentDefinitionsDb.update({ paymentDefinitionID: paymentDefinition.paymentDefinitionID }, {
    $set: paymentDefinition
  }, { upsert: true });
  if(paymentDefinition.submitted !== undefined) {
    emitEvent('payment-definition-submitted', paymentDefinition);
  } else if(paymentDefinition.transactionHash !== undefined) {
    emitEvent('payment-definition-created', paymentDefinition);
  }
};

export const markPaymentDefinitionAsConflict = async (paymentDefinitionID: string, timestamp: number) => {
  await paymentDefinitionsDb.update({ paymentDefinitionID }, { $set: { conflict: true, timestamp } });
  emitEvent('payment-definition-name-conflict', { paymentDefinitionID })
};

// ASSET INSTANCE QUERIES

export const retrieveAssetInstances = (query: object, skip: number, limit: number): Promise<IDBAssetInstance[]> => {
  return assetInstancesDb.find<IDBAssetInstance>(query, { _id: 0 }).skip(skip).limit(limit);
};

export const countAssetInstances = (query: object): Promise<number> => {
  return assetInstancesDb.count(query);
};

export const retrieveAssetInstanceByID = (assetInstanceID: string): Promise<IDBAssetInstance | null> => {
  return assetInstancesDb.findOne<IDBAssetInstance>({ assetInstanceID }, { _id: 0 });
};

export const retrieveAssetInstanceByDefinitionIDAndContentHash = (assetDefinitionID: string, contentHash: string):
  Promise<IDBAssetInstance | null> => {
  return assetInstancesDb.findOne<IDBAssetInstance>({ assetDefinitionID, contentHash }, { _id: 0 });;
};

export const upsertAssetInstance = async (assetInstance: IDBAssetInstance) => {
  await assetInstancesDb.update({ assetInstanceID: assetInstance.assetInstanceID }, {
    $set: assetInstance
  }, { upsert: true });
  if(assetInstance.submitted !== undefined) {
    emitEvent('asset-instance-submitted', assetInstance);
  } else if(assetInstance.transactionHash !== undefined) {
    emitEvent('asset-instance-created', assetInstance);
  }
};

export const setAssetInstancePrivateContent = async (assetInstanceID: string, content: object | undefined, filename: string | undefined) => {
  await assetInstancesDb.update({ assetInstanceID }, { $set: { content, filename } });
  emitEvent('private-asset-instance-content-stored', { assetInstanceID, content, filename })
};

export const markAssetInstanceAsConflict = async (assetInstanceID: string, timestamp: number) => {
  await assetInstancesDb.update({ assetInstanceID }, { $set: { conflict: true, timestamp } });
  emitEvent('asset-instance-content-conflict', { assetInstanceID });
};

export const setSubmittedAssetInstanceProperty = async (assetInstanceID: string, author: string, key: string, value: string, submitted: number,
  receipt: string | undefined) => {
  await assetInstancesDb.update({ assetInstanceID }, {
    $set: {
      [`properties.${author}.${key}.value`]: value,
      [`properties.${author}.${key}.submitted`]: submitted,
      [`properties.${author}.${key}.receipt`]: receipt,
    }
  });
  emitEvent('asset-instance-property-submitted', { assetInstanceID, key, value, submitted, receipt });
};

export const setConfirmedAssetInstanceProperty = async (assetInstanceID: string, author: string, key: string, value: string, timestamp: number,
  { blockNumber, transactionHash }: IDBBlockchainData) => {
  await assetInstancesDb.update({ assetInstanceID }, {
    $set: {
      [`properties.${author}.${key}.value`]: value,
      [`properties.${author}.${key}.history.${timestamp}`]:
        { value, timestamp, blockNumber, transactionHash }
    }
  });
  emitEvent('asset-instance-property-set', { assetInstanceID, key, value, timestamp, blockNumber, transactionHash });
};

// PAYMENT INSTANCE QUERIES

export const retrievePaymentInstances = (query: object, skip: number, limit: number): Promise<IDBPaymentInstance[]> => {
  return paymentInstancesDb.find<IDBPaymentInstance>(query, { _id: 0 }).skip(skip).limit(limit);
};

export const countPaymentInstances = (query: object): Promise<number> => {
  return paymentInstancesDb.count(query);
};

export const retrievePaymentInstanceByID = (paymentInstanceID: string): Promise<IDBPaymentInstance | null> => {
  return paymentInstancesDb.findOne<IDBPaymentInstance>({ paymentInstanceID }, { _id: 0 });
};

export const upsertPaymentInstance = async (paymentInstance: IDBPaymentInstance) => {
  await paymentInstancesDb.update({ paymentInstanceID: paymentInstance.paymentInstanceID }, {
    $set: paymentInstance
  }, { upsert: true });
  if(paymentInstance.submitted !== undefined) {
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