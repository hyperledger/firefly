import Datastore from 'nedb-promises';
import { constants, databaseCollections } from '../../lib/utils';
import path from 'path';
import { IDatabaseProvider } from '../../lib/interfaces';

const projection = { _id: 0 };
let collections: { [name: string]: Datastore } = {};

export default class NEDBProvider implements IDatabaseProvider {

  async init() {
    try {
      for (const databaseCollection of Object.values(databaseCollections)) {
        const collection = Datastore.create({
          filename: path.join(constants.DATA_DIRECTORY, `${databaseCollection.name}.json`),
          autoload: true
        });
        collection.ensureIndex({ fieldName: databaseCollection.key, unique: true });
        collections[databaseCollection.name] = collection;
      }
    } catch (err) {
      throw new Error(`Failed to initialize NEDB. ${err}`);
    }
  }

  count(collectionName: string, query: object): Promise<number> {
    return collections[collectionName].count(query);
  }

  find<T>(collectionName: string, query: object, sort: object, skip: number, limit: number): Promise<T[]> {
    return collections[collectionName].find<T>(query, projection).skip(skip).limit(limit).sort(sort);
  }

  findOne<T>(collectionName: string, query: object): Promise<T | null> {
    return collections[collectionName].findOne<T>(query, projection);
  }

  async updateOne(collectionName: string, query: object, value: object, upsert: boolean) {
    await collections[collectionName].update(query, value, { upsert });
  }

  shutDown() { }

}
