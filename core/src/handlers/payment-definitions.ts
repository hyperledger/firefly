import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IEventPaymentDefinitionCreated } from '../lib/interfaces';

export const handleGetPaymentDefinitionsRequest = (skip: number, limit: number) => {
  return database.retrievePaymentDefinitions(skip, limit);
};

export const handleGetPaymentDefinitionRequest = (paymentDefinitionID: number) => {
  return database.retrievePaymentDefinitionByID(paymentDefinitionID);
};

export const handleCreatePaymentDefinitionRequest = async (name: string, author: string, amount: number, descriptionSchema?: Object) => {
  if (await database.retrieveAssetDefinitionByName(name) !== null) {
    throw new RequestError('Asset definition name conflict', 409);
  }
  if (descriptionSchema) {
    const descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    await apiGateway.createDescribedPaymentDefinition(name, author, amount, descriptionSchemaHash);
  } else {
    await apiGateway.createPaymentDefinition(name, author, amount);
  }
  await database.upsertPaymentDefinition(name, author, descriptionSchema, amount, utils.getTimestamp(), false);
};

export const handlePaymentDefinitionCreatedEvent = async (event: IEventPaymentDefinitionCreated) => {
  if (await database.retrievePaymentDefinitionByName(event.name) !== null) {
    await database.confirmPaymentDefinition(event.name, Number(event.timestamp), Number(event.paymentDefinitionID));
  } else {
    let descriptionSchema;
    if (event.descriptionSchemaHash) {
      descriptionSchema = await ipfs.downloadJSON<Object>(utils.sha256ToIPFSHash(event.descriptionSchemaHash));
    }
    database.upsertPaymentDefinition(event.name, event.author, descriptionSchema, Number(event.amount), Number(event.timestamp), true, Number(event.paymentDefinitionID));
  }
};
