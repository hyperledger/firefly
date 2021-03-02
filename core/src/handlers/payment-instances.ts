import Ajv from 'ajv';
import { v4 as uuidV4 } from 'uuid';
import * as database from '../clients/database';
import * as ipfs from '../clients/ipfs';
import * as utils from '../lib/utils';
import * as apiGateway from '../clients/api-gateway';
import RequestError from '../lib/request-error';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IDBBlockchainData, IDBPaymentInstance, IEventPaymentInstanceCreated } from '../lib/interfaces';
import { config } from '../lib/config';

const ajv = new Ajv();

export const handleGetPaymentInstancesRequest = (query: object, skip: number, limit: number) => {
  return database.retrievePaymentInstances(query, skip, limit);
};

export const handleCountPaymentInstancesRequest = async (query: object) => {
  return { count: await database.countPaymentInstances(query) };
};

export const handleGetPaymentInstanceRequest = async (paymentInstanceID: string) => {
  const assetInstance = await database.retrievePaymentInstanceByID(paymentInstanceID);
  if (assetInstance === null) {
    throw new RequestError('Payment instance not found', 404);
  }
  return assetInstance;
};

export const handleCreatePaymentInstanceRequest = async (author: string, paymentDefinitionID: string,
  recipient: string, description: object | undefined, amount: number, participants: string[] | undefined, sync: boolean) => {
  const paymentDefinition = await database.retrievePaymentDefinitionByID(paymentDefinitionID);
  if (paymentDefinition === null) {
    throw new RequestError('Unknown payment definition', 400);
  }
  if (paymentDefinition.transactionHash === undefined) {
    throw new RequestError('Payment definition transaction must be mined', 400);
  }
  if(config.protocol === 'ethereum' && participants !== undefined) {
    throw new RequestError('Participants not supported in Ethereum', 400);
  }
  if(config.protocol === 'corda') {
    // validate participants are registered members
    if(participants !== undefined) {
      for(const participant  of participants) {
        if (await database.retrieveMemberByAddress(participant) === null) {
          throw new RequestError('One or more participants are not registered', 400);
        }
      }
    } else {
      throw new RequestError('Missing payment participants', 400);
    }
  }
  let descriptionHash: string | undefined;
  if (paymentDefinition.descriptionSchema) {
    if (!description) {
      throw new RequestError('Missing payment description', 400);
    }
    if (!ajv.validate(paymentDefinition.descriptionSchema, description)) {
      throw new RequestError('Description does not conform to payment definition schema', 400);
    }
    descriptionHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(description)));
  }
  const paymentInstanceID = uuidV4();
  const timestamp = utils.getTimestamp();
  let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
  if (descriptionHash) {
    apiGatewayResponse = await apiGateway.createDescribedPaymentInstance(paymentInstanceID,
      paymentDefinitionID, author, recipient, amount, descriptionHash, participants,sync);
  } else {
    apiGatewayResponse = await apiGateway.createPaymentInstance(paymentInstanceID,
      paymentDefinitionID, author, recipient, amount, participants, sync);
  }
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.upsertPaymentInstance({
    paymentInstanceID,
    author,
    paymentDefinitionID: paymentDefinition.paymentDefinitionID,
    descriptionHash,
    description,
    recipient,
    participants,
    amount,
    receipt,
    submitted: timestamp
  });
  return paymentInstanceID;
};

export const handlePaymentInstanceCreatedEvent = async (event: IEventPaymentInstanceCreated, { blockNumber, transactionHash }: IDBBlockchainData) => {
  const eventPaymentInstanceID = utils.hexToUuid(event.paymentInstanceID);
  const dbPaymentInstance = await database.retrievePaymentInstanceByID(eventPaymentInstanceID);
  if (dbPaymentInstance !== null && dbPaymentInstance.transactionHash !== undefined) {
    throw new Error(`Duplicate payment instance ID`);
  }
  const paymentDefinition = await database.retrievePaymentDefinitionByID(utils.hexToUuid(event.paymentDefinitionID));
  if (paymentDefinition === null) {
    throw new Error('Uknown payment definition');
  }
  if (config.protocol === 'ethereum' && paymentDefinition.transactionHash === undefined) {
    throw new Error('Payment definition transaction must be mined');
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
  let paymentInstanceDB: IDBPaymentInstance = {
    paymentInstanceID: eventPaymentInstanceID,
    author: event.author,
    paymentDefinitionID: paymentDefinition.paymentDefinitionID,
    descriptionHash: event.descriptionHash,
    description,
    recipient: event.recipient,
    amount: Number(event.amount),
    timestamp: Number(event.timestamp),
    blockNumber,
    transactionHash
  };
  if(config.protocol === 'corda') {
    paymentInstanceDB.participants = event.participants;
  }
  database.upsertPaymentInstance(paymentInstanceDB);
};
