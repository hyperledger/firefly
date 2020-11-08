import { v4 as uuidV4 } from 'uuid';
import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IEventPaymentDefinitionCreated } from '../lib/interfaces';

export const handleGetPaymentDefinitionsRequest = (skip: number, limit: number) => {
  return database.retrievePaymentDefinitions(skip, limit);
};

export const handleGetPaymentDefinitionRequest = async (paymentDefinitionID: string) => {
  const paymentDefinition = await database.retrievePaymentDefinitionByID(paymentDefinitionID);
  if (paymentDefinition === null) {
    throw new RequestError('Payment definition not found', 404);
  }
  return paymentDefinition;
};

export const handleCreatePaymentDefinitionRequest = async (name: string, author: string, descriptionSchema: Object | undefined, sync: boolean) => {
  if (await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Payment definition name conflict', 409);
  }
  const assetDefinitionID = uuidV4();
  if (descriptionSchema) {
    const descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    await database.upsertPaymentDefinition(assetDefinitionID, name, author, descriptionSchemaHash, descriptionSchema, utils.getTimestamp(), false);
    await apiGateway.createDescribedPaymentDefinition(name, author, descriptionSchemaHash, sync);
  } else {
    await database.upsertPaymentDefinition(assetDefinitionID, name, author, undefined, undefined, utils.getTimestamp(), false);
    await apiGateway.createPaymentDefinition(name, author, sync);
  }
  return assetDefinitionID;
};

export const handlePaymentDefinitionCreatedEvent = async (event: IEventPaymentDefinitionCreated) => {
  const paymentDefinitionID = utils.hexToUuid(event.paymentDefinitionID);
  const dbPaymentDefinitionByID = await database.retrievePaymentDefinitionByID(paymentDefinitionID);
  if (dbPaymentDefinitionByID !== null) {
    if (dbPaymentDefinitionByID.confirmed) {
      throw new Error(`Payment definition ID conflict ${paymentDefinitionID}`);
    }
  } else {
    const dbpaymentDefinitionByName = await database.retrievePaymentDefinitionByName(event.name);
    if (dbpaymentDefinitionByName !== null) {
      if (dbpaymentDefinitionByName.confirmed) {
        throw new Error(`Payment definition name conflict ${event.name}`);
      } else {
        await database.markPaymentDefinitionAsConflict(dbpaymentDefinitionByName.paymentDefinitionID, Number(event.timestamp));
      }
    }
  }
  let descriptionSchema;
  if (event.descriptionSchemaHash) {
    if (event.descriptionSchemaHash === dbPaymentDefinitionByID?.descriptionSchemaHash) {
      descriptionSchema = dbPaymentDefinitionByID?.descriptionSchema
    } else {
      descriptionSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.descriptionSchemaHash));
    }
  }
  database.upsertPaymentDefinition(paymentDefinitionID, event.name, event.author, event.descriptionSchemaHash, descriptionSchema, Number(event.timestamp), true);
};
