import Datastore from 'nedb-promises';
import { constants, databaseCollectionIndexFields } from '../../lib/utils';
import path from 'path';
import { databaseCollectionName, IDatabaseProvider } from '../../lib/interfaces';

const projection = { _id: 0 };
let collections: { [name: string]: Datastore } = {};

export default class NEDBProvider implements IDatabaseProvider {

  async init() {
    try {
      for (const [collectionName, indexField] of Object.entries(databaseCollectionIndexFields)) {
        const collection = Datastore.create({
          filename: path.join(constants.DATA_DIRECTORY, `${collectionName}.json`),
          autoload: true
        });
        collection.ensureIndex({ fieldName: indexField, unique: true });
        collections[collectionName] = collection;
      }
    } catch (err) {
      throw new Error(`Failed to initialize NEDB. ${err}`);
    }
  }

  count(collectionName: databaseCollectionName, query: object): Promise<number> {
    return collections[collectionName].count(query);
  }

  find<T>(collectionName: databaseCollectionName, query: object, sort: object, skip: number, limit: number): Promise<T[]> {
    return collections[collectionName].find<T>(query, projection).skip(skip).limit(limit).sort(sort);
  }

  findOne<T>(collectionName: databaseCollectionName, query: object): Promise<T | null> {
    return collections[collectionName].findOne<T>(query, projection);
  }

  async updateOne(collectionName: databaseCollectionName, query: object, value: object, upsert: boolean) {
    await collections[collectionName].update(query, value, { upsert });
  }

  shutDown() { }

}
