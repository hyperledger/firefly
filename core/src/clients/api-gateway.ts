import { config } from '../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse } from '../lib/interfaces';
import * as ethrereumGateay from './gateway-providers/ethereum';
import * as cordaGateway from './gateway-providers/corda';

// Member APIs

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethrereumGateay.upsertMember(address, name, app2appDestination, docExchangeDestination, sync);
};


// Asset definition APIs

export const createAssetDefinition = async (author: string, assetDefinitionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethrereumGateay.createAssetDefinition(author, assetDefinitionHash, sync);
};


// Payment definition APIs

export const createDescribedPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string,
  descriptionSchemaHash: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethrereumGateay.createDescribedPaymentDefinition(paymentDefinitionID, name, author, descriptionSchemaHash, sync);
};

export const createPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethrereumGateay.createPaymentDefinition(paymentDefinitionID, name, author, sync);
};


// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  descriptionHash: string, contentHash: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, descriptionHash, contentHash, participants);
    case 'ethereum':
      return ethrereumGateay.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, sync);
  }
};

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  contentHash: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createAssetInstance(assetInstanceID, assetDefinitionID, contentHash, participants);
    case 'ethereum':
      return ethrereumGateay.createAssetInstance(assetInstanceID, assetDefinitionID, author, contentHash, sync);
  }
};

export const createAssetInstanceBatch = async (batchHash: string, author: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createAssetInstanceBatch(batchHash, participants);
    case 'ethereum':
      return ethrereumGateay.createAssetInstanceBatch(batchHash, author, sync);
  }
}

export const setAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string,
  participants: string[] | undefined, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.setAssetInstanceProperty(assetDefinitionID, assetInstanceID, key, value, participants);
    case 'ethereum':
      return ethrereumGateay.setAssetInstanceProperty(assetDefinitionID, assetInstanceID, author, key, value, sync);
  }
};


// Payment instance APIs

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, descriptionHash: string, participants: string[] | undefined, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createDescribedPaymentInstance(paymentInstanceID, paymentDefinitionID, recipient, amount, descriptionHash, participants);
    case 'ethereum':
      return ethrereumGateay.createDescribedPaymentInstance(paymentInstanceID, paymentDefinitionID, author, recipient, amount, descriptionHash, sync);
  }
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, participants: string[] | undefined, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createPaymentInstance(paymentInstanceID, paymentDefinitionID, recipient, amount, participants);
    case 'ethereum':
      return ethrereumGateay.createPaymentInstance(paymentInstanceID, paymentDefinitionID, author, recipient, amount, sync);
  }
};
