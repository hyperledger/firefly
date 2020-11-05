import Datastore from 'nedb-promises';
import { constants } from '../lib/utils';
import path from 'path';
import { IDBAssetDefinition, IDBAssetInstance, IDBMember, IDBPaymentDefinition } from '../lib/interfaces';

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
assetDefinitionsDb.ensureIndex({ fieldName: 'name', unique: true });
paymentDefinitionsDb.ensureIndex({ fieldName: 'name', unique: true });
assetInstancesDb.ensureIndex({ fieldName: 'assetInstanceID', unique: true });
paymentInstancesDb.ensureIndex({ fieldName: 'paymentInstanceID', unique: true });

// Member queries

export const retrieveMemberByAddress = (address: string): Promise<IDBMember | null> => {
  return membersDb.findOne<IDBMember>({ address }, { _id: 0 });
};

export const retrieveMembers = (skip: number, limit: number, owned: boolean): Promise<IDBMember[]> => {
  let query: any = {};
  if (owned) {
    query.owned = true;
  }
  return membersDb.find<IDBMember>(query, { _id: 0 }).skip(skip).limit(limit);
};

export const upsertMember = (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, timestamp: number, confirmed: boolean, owned: boolean): Promise<number> => {
  return membersDb.update({ address }, { $set: { address, name, app2appDestination, docExchangeDestination, timestamp, confirmed, owned } }, { upsert: true });
};

export const confirmMember = (address: string, timestamp: number) => {
  return membersDb.update({ address }, { $set: { timestamp, confirmed: true } });
};

// Asset definition queries

export const retrieveAssetDefinitions = (skip: number, limit: number): Promise<IDBAssetDefinition[]> => {
  return assetDefinitionsDb.find<IDBAssetDefinition>({}, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 })
};

export const retrieveAssetDefinitionByID = (assetDefinitionID: number): Promise<IDBAssetDefinition | null> => {
  return assetDefinitionsDb.findOne<IDBAssetDefinition>({ assetDefinitionID }, { _id: 0 });
};

export const retrieveAssetDefinitionByName = (name: string): Promise<IDBAssetDefinition | null> => {
  return assetDefinitionsDb.findOne<IDBAssetDefinition>({ name }, { _id: 0 });
};

export const upsertAssetDefinition = (name: string, author: string, isContentPrivate: boolean, descriptionSchema: Object | undefined, contentSchema: Object | undefined, timestamp: number, confirmed: boolean, assetDefinitionID?: number) => {
  return assetDefinitionsDb.update({ name }, { $set: { name, author, isContentPrivate, descriptionSchema, contentSchema, timestamp, confirmed, assetDefinitionID } }, { upsert: true });
};

export const confirmAssetDefinition = (name: string, timestamp: number, assetDefinitionID: number) => {
  return assetDefinitionsDb.update({ name }, { $set: { timestamp, confirmed: true, assetDefinitionID } });
};

// Payment definition queries

export const retrievePaymentDefinitions = (skip: number, limit: number): Promise<IDBPaymentDefinition[]> => {
  return paymentDefinitionsDb.find<IDBPaymentDefinition>({}, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 })
};

export const retrievePaymentDefinitionByID = (paymentDefinitionID: number): Promise<IDBPaymentDefinition | null> => {
  return paymentDefinitionsDb.findOne<IDBPaymentDefinition>({ paymentDefinitionID }, { _id: 0 });
};

export const retrievePaymentDefinitionByName = (name: string): Promise<IDBPaymentDefinition | null> => {
  return paymentDefinitionsDb.findOne<IDBPaymentDefinition>({ name }, { _id: 0 });
};

export const upsertPaymentDefinition = (name: string, author: string, descriptionSchema: Object | undefined, amount: number, timestamp: number, confirmed: boolean, paymentDefinitionID?: number) => {
  return paymentDefinitionsDb.update({ name }, { $set: { name, author, descriptionSchema, amount, timestamp, confirmed, paymentDefinitionID } }, { upsert: true });
};

export const confirmPaymentDefinition = (name: string, timestamp: number, paymentDefinitionID: number) => {
  return paymentDefinitionsDb.update({ name }, { $set: { timestamp, confirmed: true, paymentDefinitionID } });
};

// Asset instance queries

export const retrieveAssetInstances = (skip: number, limit: number): Promise<IDBAssetInstance[]> => {
  return assetInstancesDb.find<IDBAssetInstance>({}, { _id: 0 }).skip(skip).limit(limit);
};

export const retrieveAssetInstanceByID = (assetInstanceID: string): Promise<IDBAssetInstance | null> => {
  return assetInstancesDb.findOne<IDBAssetInstance>({ assetInstanceID }, { _id: 0 });
};

export const upsertAssetInstance = (assetInstanceID: string, author: string, assetDefinitionID: number, description: Object | undefined, contentHash: string, content: Object | undefined, confirmed: boolean, timestamp: number) => {
  return assetInstancesDb.update({ assetInstanceID }, { $set: { author, assetDefinitionID, description, contentHash, content, confirmed, timestamp, assetInstanceID } }, { upsert: true });
};

export const confirmAssetInstance = (assetInstanceID: string, timestamp: number) => {
  return assetInstancesDb.update({ assetInstanceID }, { timestamp, confirmed: true });
}