import { v4 as uuidV4 } from 'uuid';
import Ajv from 'ajv';
import * as utils from '../lib/utils';
import * as ipfs from '../clients/ipfs';
import * as apiGateway from '../clients/api-gateway';
import * as database from '../clients/database';
import RequestError from '../lib/request-error';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IDBBlockchainData, IDBPaymentDefinition, IEventPaymentDefinitionCreated } from '../lib/interfaces';
import { config } from '../lib/config';

const ajv = new Ajv();

export const handleGetPaymentDefinitionsRequest = (query: object, skip: number, limit: number) => {
  return database.retrievePaymentDefinitions(query, skip, limit);
};

export const handleCountPaymentDefinitionsRequest = async (query: object) => {
  return { count: await database.countPaymentDefinitions(query) };
};

export const handleGetPaymentDefinitionRequest = async (paymentDefinitionID: string) => {
  const paymentDefinition = await database.retrievePaymentDefinitionByID(paymentDefinitionID);
  if (paymentDefinition === null) {
    throw new RequestError('Payment definition not found', 404);
  }
  return paymentDefinition;
};

export const handleCreatePaymentDefinitionRequest = async (name: string, author: string, descriptionSchema: Object | undefined, participants: string[] | undefined, sync: boolean) => {
  if(descriptionSchema !== undefined && !ajv.validateSchema(descriptionSchema)) {
    throw new RequestError('Invalid description schema', 400);
  }
  if (await database.retrievePaymentDefinitionByName(name) !== null) {
    throw new RequestError('Payment definition name conflict', 409);
  }
  if(config.protocol === 'corda') {
    //check participants are valid addresses of registered members
    if(participants) {
      for(var participant  of participants) {
        if (await database.retrieveMemberByAddress(participant) === null) {
          throw new RequestError(`Participant ${participant} is not a registered member`, 409);
        }
      }
    } else {
      throw new RequestError(`Missing payment definition participants`, 400);
    }
  }
  let descriptionSchemaHash: string | undefined;
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
  const timestamp = utils.getTimestamp();

  const paymentDefinitionID = uuidV4();
  if (descriptionSchema) {
    descriptionSchemaHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(descriptionSchema)));
    apiGatewayResponse = await apiGateway.createDescribedPaymentDefinition(paymentDefinitionID, name, author, descriptionSchemaHash, participants, sync);
  } else {
    apiGatewayResponse = await apiGateway.createPaymentDefinition(paymentDefinitionID, name, author, participants, sync);
  }
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  var paymentDefinitionDB: IDBPaymentDefinition = {
    paymentDefinitionID,
    name,
    author,
    descriptionSchemaHash,
    descriptionSchema,
    submitted: timestamp,
    receipt
  };
  if(config.protocol === 'corda') {
    paymentDefinitionDB.participants = participants;
  }
  await database.upsertPaymentDefinition(paymentDefinitionDB);
  return paymentDefinitionID;
};

export const handlePaymentDefinitionCreatedEvent = async (event: IEventPaymentDefinitionCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {
  const paymentDefinitionID = utils.hexToUuid(event.paymentDefinitionID);
  const dbPaymentDefinitionByID = await database.retrievePaymentDefinitionByID(paymentDefinitionID);
  if (dbPaymentDefinitionByID !== null) {
    if (dbPaymentDefinitionByID.transactionHash !== undefined) {
      throw new Error(`Payment definition ID conflict ${paymentDefinitionID}`);
    }
  } else {
    const dbpaymentDefinitionByName = await database.retrievePaymentDefinitionByName(event.name);
    if (dbpaymentDefinitionByName !== null) {
      if (dbpaymentDefinitionByName.transactionHash !== undefined) {
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
  database.upsertPaymentDefinition({
    paymentDefinitionID,
    author: event.author,
    name: event.name,
    descriptionSchemaHash: event.descriptionSchemaHash,
    descriptionSchema,
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  });
};
