import Datastore from 'nedb-promises';
import { constants } from '../lib/utils';
import path from 'path';
import { IDBMember } from '../lib/interfaces';

const membersDb = Datastore.create({
  filename: path.join(constants.DATA_DIRECTORY, constants.MEMBERS_DATABASE_FILE_NAME),
  autoload: true
});

membersDb.ensureIndex({ fieldName: 'address', unique: true });

export const retrieveMember = (address: string): Promise<IDBMember | null> => {
  return membersDb.findOne<IDBMember>({ address }, { _id: 0 });
};

export const retrieveMembers = (skip: number, limit: number): Promise<IDBMember[]> => {
  return membersDb.find<IDBMember>({}, { _id: 0 }).skip(skip).limit(limit);
};

export const upsertMember = (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, timestamp: number, confirmed: boolean, owned: boolean): Promise<number> => {
  return membersDb.update({ address }, { address, name, app2appDestination, docExchangeDestination, timestamp, confirmed, owned }, { upsert: true });
};
