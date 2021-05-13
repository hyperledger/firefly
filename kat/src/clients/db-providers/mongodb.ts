// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { Db, MongoClient } from 'mongodb';
import { config } from '../../lib/config';
import { databaseCollectionName, IDatabaseProvider, indexes } from '../../lib/interfaces';
import { databaseCollectionIndexes } from '../../lib/utils';

let db: Db;
let mongoClient: MongoClient;

export default class MongoDBProvider implements IDatabaseProvider {

  async init() {
    try {
      mongoClient = await MongoClient.connect(config.mongodb.connectionUrl,
        { useNewUrlParser: true, useUnifiedTopology: true, ignoreUndefined: true });
      db = mongoClient.db(config.mongodb.databaseName);
      for (const [collectionName, indexes] of Object.entries(databaseCollectionIndexes)) {
        this.createCollection(collectionName, indexes);
      }
    } catch (err) {
      throw new Error(`Failed to connect to Mongodb. ${err}`);
    }
  }

  async createCollection(collectionName: string, indexes: indexes) {
    try {
      for (const index of indexes) {
        const fields: { [f: string]: number } = {};
        for (const field of index.fields) {
          fields[field] = 1; // all ascending currently
        }
        db.collection(collectionName).createIndex(fields, { unique: !!index.unique });
      }
    } catch (err) {
      throw new Error(`Failed to create collection. ${err}`);
    }
  }

  count(collectionName: databaseCollectionName, query: object): Promise<number> {
    return db.collection(collectionName).find(query).count();
  }

  find<T>(collectionName: databaseCollectionName, query: object, sort: object, skip: number, limit: number): Promise<T[]> {
    return db.collection(collectionName).find<T>(query, { projection: { _id: 0 } }).sort(sort).skip(skip).limit(limit).toArray();
  }

  findOne<T>(collectionName: databaseCollectionName, query: object): Promise<T | null> {
    return db.collection(collectionName).findOne<T>(query, { projection: { _id: 0 } });
  }

  async updateOne(collectionName: databaseCollectionName, query: object, value: object, upsert: boolean) {
    await db.collection(collectionName).updateOne(query, value, { upsert });
  }

  shutDown() {
    mongoClient.close();
  }

}
