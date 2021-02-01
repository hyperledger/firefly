// This is used to move asset instances from the "asset-instances" mongodb collection into their
// assetDefinitionID specific collection. It does not delete any collections/documents.
//
// It will need to be run on each mongo DB
// Example usage: node asset-instances-migration.js -c mongodb://localhost:27017 -n shorsher-org-2

const MongoClient = require('mongodb').MongoClient;
const argv = require('yargs')
  .option('connectionUrl', { alias: 'c', type: 'string' })
  .option('dbName', { alias: 'n', type: 'string' })
  .argv;

const ASSET_DEFINITIONS = 'asset-definitions';
const ASSET_INSTANCES = 'asset-instances';

let db;
let client;

const log = (...stuff) => {
  console.log(`${new Date().toISOString()}: `, ...stuff);
}

async function initDB() {
  try {
    client = await MongoClient.connect(argv.connectionUrl,
      { useNewUrlParser: true, useUnifiedTopology: true, ignoreUndefined: true });
    db = client.db(argv.dbName);
  } catch (err) {
    throw new Error(`Failed to connect to Mongodb. ${err}`);
  }
}

async function createDefinitionCollections() {
  const collection = db.collection(ASSET_DEFINITIONS);
  const definitions = await collection.find({}).toArray();
  for (const definition of definitions) {
    await db.collection(`asset-instance-${definition.assetDefinitionID}`).createIndex('assetInstanceID', { unique: true });
  }
}

async function migrateAssetInstances() {
  let cursor = db.collection(ASSET_INSTANCES).find()
  while (await cursor.hasNext()) {
    let doc = await cursor.next();
    await db.collection(`asset-instance-${doc.assetDefinitionID}`).updateOne({ assetInstanceID: doc.assetInstanceID }, { $set: doc}, { upsert: true });
    log(`migrated ${doc.assetInstanceID} to asset-instance-${doc.assetDefinitionID}`);
  }
};

async function run() {
  await initDB();
  await createDefinitionCollections();
  await migrateAssetInstances();
  client.close();
}

run()
  .catch(err => {
    console.error(err.stack);
    process.exit(1);
  })
