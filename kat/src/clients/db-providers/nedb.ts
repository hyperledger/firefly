import Datastore from 'nedb-promises';
import path from 'path';
import { databaseCollectionName, IDatabaseProvider, indexes } from '../../lib/interfaces';
import { constants, databaseCollectionIndexes } from '../../lib/utils';

const projection = { _id: 0 };
let collections: { [name: string]: Datastore } = {};

export default class NEDBProvider implements IDatabaseProvider {

  async init() {
    try {
      for (const [collectionName, indexes] of Object.entries(databaseCollectionIndexes)) {
        this.createCollection(collectionName, indexes);
      }
    } catch (err) {
      throw new Error(`Failed to initialize NEDB. ${err}`);
    }
  }

  async createCollection(collectionName: string, indexes: indexes) {
    const collection = Datastore.create({
      filename: path.join(constants.DATA_DIRECTORY, `${collectionName}.json`),
      autoload: true
    });
    for (const index of indexes) {
      // No compound indexes here
      for (let fieldName of index.fields) {
        collection.ensureIndex({ fieldName, unique: !!index.unique });
      }
    }
    collections[collectionName] = collection;
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
