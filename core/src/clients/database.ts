import Datastore from 'nedb-promises';
import { constants } from '../lib/utils';
import path from 'path';
import { IDBAssetDefinition, IDBAssetInstance, IDBBlockchainData, IDBMember, IDBPaymentDefinition, IDBPaymentInstance } from '../lib/interfaces';

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

// Member queries

export const retrieveMemberByAddress = (address: string): Promise<IDBMember | null> => {
  return membersDb.findOne<IDBMember>({ address }, { _id: 0 });
};

export const retrieveMembers = (skip: number, limit: number, owned: boolean): Promise<IDBMember[]> => {
  let query: any = {};
  if (owned) {
    query.owned = true;
  }
  return membersDb.find<IDBMember>(query, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 });
};

// export const upsertMemberFromRequest = (address: string, name: string, assetTrailInstanceID: string, app2appDestination: string,
//   docExchangeDestination: string, receipt: string | undefined, timestamp: number) => {
//   return membersDb.update({ address }, {
//     $set: {
//       address, name, assetTrailInstanceID, app2appDestination, docExchangeDestination, timestamp, receipt
//     }
//   }, { upsert: true });
// };

// export const upsertMemberFromEvent = (address: string, name: string, assetTrailInstanceID: string, app2appDestination: string,
//   docExchangeDestination: string, timestamp: number, blockchainData: IDBBlockchainData | undefined) => {
//   return membersDb.update({ address }, {
//     $set: {
//       address, name, assetTrailInstanceID, app2appDestination, docExchangeDestination, timestamp, blockchainData
//     }
//   }, { upsert: true });
// };

export const upsertMember = (member: IDBMember) => {
  return membersDb.update({ address: member.address }, {
    $set: member
  }, { upsert: true });
};

export const isMemberOwned = async (address: string): Promise<boolean> => {
  return (await membersDb.count({ address, owned: true })) === 1;
};

// Asset definition queries

export const retrieveAssetDefinitions = (skip: number, limit: number): Promise<IDBAssetDefinition[]> => {
  return assetDefinitionsDb.find<IDBAssetDefinition>({}, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 })
};

export const retrieveAssetDefinitionByID = (assetDefinitionID: string): Promise<IDBAssetDefinition | null> => {
  return assetDefinitionsDb.findOne<IDBAssetDefinition>({ assetDefinitionID }, { _id: 0 });
};

export const retrieveAssetDefinitionByName = (name: string): Promise<IDBAssetDefinition | null> => {
  return assetDefinitionsDb.findOne<IDBAssetDefinition>({ name }, { _id: 0 });
};

export const upsertAssetDefinition = (assetDefinitionID: string, name: string, author: string, isContentPrivate: boolean,
  isContentUnique: boolean, descriptionSchemaHash: string | undefined, descriptionSchema: Object | undefined,
  contentSchemaHash: string | undefined, contentSchema: Object | undefined,
  timestamp: number, confirmed: boolean, blockchainData: IDBBlockchainData | undefined) => {
  return assetDefinitionsDb.update({ assetDefinitionID }, {
    $set: {
      assetDefinitionID, name, author, isContentPrivate, isContentUnique, descriptionSchemaHash,
      descriptionSchema, contentSchemaHash, contentSchema, timestamp, confirmed, blockchainData
    }
  }, { upsert: true });
};

export const markAssetDefinitionAsConflict = (assetDefinitionID: string, timestamp: number) => {
  return assetDefinitionsDb.update({ assetDefinitionID }, { $set: { timestamp, conflict: true } });
};

// Payment definition queries

export const retrievePaymentDefinitions = (skip: number, limit: number): Promise<IDBPaymentDefinition[]> => {
  return paymentDefinitionsDb.find<IDBPaymentDefinition>({}, { _id: 0 }).skip(skip).limit(limit).sort({ name: 1 })
};

export const retrievePaymentDefinitionByID = (paymentDefinitionID: string): Promise<IDBPaymentDefinition | null> => {
  return paymentDefinitionsDb.findOne<IDBPaymentDefinition>({ paymentDefinitionID }, { _id: 0 });
};

export const retrievePaymentDefinitionByName = (name: string): Promise<IDBPaymentDefinition | null> => {
  return paymentDefinitionsDb.findOne<IDBPaymentDefinition>({ name }, { _id: 0 });
};

export const upsertPaymentDefinition = (paymentDefinitionID: string, name: string, author: string, descriptionSchemaHash: string | undefined,
  descriptionSchema: Object | undefined, timestamp: number, confirmed: boolean, blockchainData: IDBBlockchainData | undefined) => {
  return paymentDefinitionsDb.update({ paymentDefinitionID }, {
    $set: {
      paymentDefinitionID, name, author, descriptionSchemaHash, descriptionSchema,
      timestamp, confirmed, blockchainData
    }
  }, { upsert: true });
};

export const markPaymentDefinitionAsConflict = (paymentDefinitionID: string, timestamp: number) => {
  return paymentDefinitionsDb.update({ paymentDefinitionID }, { $set: { conflict: true, timestamp } });
};

// Asset instance queries

export const retrieveAssetInstances = (skip: number, limit: number): Promise<IDBAssetInstance[]> => {
  return assetInstancesDb.find<IDBAssetInstance>({}, { _id: 0 }).skip(skip).limit(limit);
};

export const retrieveAssetInstanceByID = (assetInstanceID: string): Promise<IDBAssetInstance | null> => {
  return assetInstancesDb.findOne<IDBAssetInstance>({ assetInstanceID }, { _id: 0 });
};

export const retrieveAssetInstanceByDefinitionIDAndContentHash = (assetDefinitionID: string, contentHash: string): Promise<IDBAssetInstance | null> => {
  return assetInstancesDb.findOne<IDBAssetInstance>({ assetDefinitionID, contentHash }, { _id: 0 });;
};

export const upsertAssetInstance = (assetInstanceID: string, author: string, assetDefinitionID: string, descriptionHash: string | undefined,
  description: Object | undefined, contentHash: string, content: Object | undefined, confirmed: boolean, timestamp: number,
  blockchainData: IDBBlockchainData | undefined) => {
  return assetInstancesDb.update({ assetInstanceID }, {
    $set: {
      assetInstanceID, author, assetDefinitionID, descriptionHash, description, contentHash, content,
      confirmed, blockchainData, timestamp
    }
  }, { upsert: true });
};

export const markAssetInstanceAsConflict = (assetInstanceID: string, timestamp: number) => {
  return assetInstancesDb.update({ assetInstanceID }, { $set: { conflict: true, timestamp } });
};

export const setAssetInstanceProperty = (assetInstanceID: string, author: string, key: string, value: string, confirmed: boolean, timestamp: number, blockchainData: IDBBlockchainData | undefined) => {
  return assetInstancesDb.update({ assetInstanceID }, { $set: { [`properties.${author}.${key}`]: { value, confirmed, timestamp, blockchainData } } });
};

// Payment instance queries

export const retrievePaymentInstances = (skip: number, limit: number): Promise<IDBPaymentInstance[]> => {
  return paymentInstancesDb.find<IDBPaymentInstance>({}, { _id: 0 }).skip(skip).limit(limit);
};

export const retrievePaymentInstanceByID = (paymentInstanceID: string): Promise<IDBPaymentInstance | null> => {
  return paymentInstancesDb.findOne<IDBPaymentInstance>({ paymentInstanceID }, { _id: 0 });
};

export const upsertPaymentInstance = (paymentInstanceID: string, author: string, paymentDefinitionID: string, descriptionHash: string | undefined,
  description: Object | undefined, recipient: string, amount: number, confirmed: boolean, timestamp: number, blockchainData: IDBBlockchainData | undefined) => {
  return paymentInstancesDb.update({ paymentInstanceID }, {
    $set: {
      paymentInstanceID, author, paymentDefinitionID, descriptionHash, description,
      recipient, amount, confirmed, blockchainData, timestamp
    }
  }, { upsert: true });
};
