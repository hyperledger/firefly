import Ajv from 'ajv';
import { v4 as uuidV4 } from 'uuid';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as apiGateway from '../clients/api-gateway';
import RequestError from '../lib/request-error';
import { IDBBlockchainData, IEventPaymentInstanceCreated } from '../lib/interfaces';

const ajv = new Ajv();

export const handleGetPaymentInstancesRequest = (skip: number, limit: number) => {
  return database.retrievePaymentInstances(skip, limit);
};

export const handleGetPaymentInstanceRequest = async (paymentInstanceID: string) => {
  const assetInstance = await database.retrievePaymentInstanceByID(paymentInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Payment instance not found', 404);
  }
  return assetInstance;
};

export const handleCreatePaymentInstanceRequest = async (author: string, paymentDefinitionID: string,
  recipient: string, description: object | undefined, amount: number, sync: boolean) => {
  const paymentInstanceID = uuidV4();
  let descriptionHash: string | undefined;
  const paymentDefinition = await database.retrievePaymentDefinitionByID(paymentDefinitionID);
  if (paymentDefinition === null) {
    throw new RequestError('Unknown payment definition', 400);
  }
  if (!paymentDefinition.confirmed) {
    throw new RequestError('Payment definition must be confirmed', 400);
  }
  if (paymentDefinition.descriptionSchema) {
    if (!description) {
      throw new RequestError('Missing payment description', 400);
    }
    if (!ajv.validate(paymentDefinition.descriptionSchema, description)) {
      throw new RequestError('Description does not conform to payment definition schema', 400);
    }
    descriptionHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(description)));
  }
  await database.upsertPaymentInstance(paymentInstanceID, author, paymentDefinition.paymentDefinitionID,
    descriptionHash, description, recipient, amount, false, utils.getTimestamp(), undefined);
  if (descriptionHash) {
    await apiGateway.createDescribedPaymentInstance(utils.uuidToHex(paymentInstanceID),
      utils.uuidToHex(paymentDefinitionID), author, recipient, descriptionHash, sync);
  } else {
    await apiGateway.createPaymentInstance(utils.uuidToHex(paymentInstanceID),
      utils.uuidToHex(paymentDefinitionID), author, recipient, sync);
  }
  return paymentInstanceID;
};

export const handlePaymentInstanceCreatedEvent = async (event: IEventPaymentInstanceCreated, blockchainData: IDBBlockchainData) => {
  const eventPaymentInstanceID = utils.hexToUuid(event.paymentInstanceID);
  const dbPaymentInstance = await database.retrievePaymentInstanceByID(eventPaymentInstanceID);
  if (dbPaymentInstance !== null && dbPaymentInstance.confirmed) {
    throw new Error(`Duplicate payment instance ID`);
  }
  const paymentDefinition = await database.retrievePaymentDefinitionByID(utils.hexToUuid(event.paymentDefinitionID));
  if (paymentDefinition === null) {
    throw new Error('Uknown payment definition');
  }
  if (!paymentDefinition.confirmed) {
    throw new Error('Unconfirmed payment definition');
  }
  let description: Object | undefined = undefined;
  if (paymentDefinition.descriptionSchema) {
    if (event.descriptionHash) {
      if (event.descriptionHash === dbPaymentInstance?.descriptionHash) {
        description = dbPaymentInstance.description;
      } else {
        description = await ipfs.downloadJSON(utils.sha256ToIPFSHash(event.descriptionHash));
        if (!ajv.validate(paymentDefinition.descriptionSchema, description)) {
          throw new Error('Description does not conform to schema');
        }
      }
    } else {
      throw new Error('Missing payment instance description');
    }
  }
  database.upsertPaymentInstance(eventPaymentInstanceID, event.author,
    utils.hexToUuid(event.paymentDefinitionID),
    event.descriptionHash, description, event.recipient, Number(event.amount), true,
    Number(event.timestamp), blockchainData);
};
