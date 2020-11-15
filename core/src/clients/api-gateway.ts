import axios from 'axios';
import { config } from '../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse } from '../lib/interfaces';
import * as utils from '../lib/utils';

// Member APIs

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/registerMember?kld-from=${address}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      name,
      assetTrailInstanceID: config.assetTrailInstanceID,
      app2appDestination,
      docExchangeDestination
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
};

// Asset definition APIs

export const createDescribedStructuredAssetDefinition = async (assetDefinitionID: string, name: string, author: string,
  isContentPrivate: boolean, isContentUnique: boolean, descriptionSchemaHash: string, contentSchemaHash: string,
  sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedStructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      assetDefinitionID: utils.uuidToHex(assetDefinitionID),
      name,
      isContentPrivate,
      isContentUnique,
      descriptionSchemaHash,
      contentSchemaHash
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
}

export const createDescribedUnstructuredAssetDefinition = async (assetDefinitionID: string, name: string, author: string,
  isContentPrivate: boolean, isContentUnique: boolean, descriptionSchemaHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedUnstructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      assetDefinitionID: utils.uuidToHex(assetDefinitionID),
      name, isContentPrivate,
      isContentUnique,
      descriptionSchemaHash
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
}

export const createStructuredAssetDefinition = async (assetDefinitionID: string, name: string, author: string,
  isContentPrivate: boolean, isContentUnique: boolean, contentSchemaHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createStructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      assetDefinitionID: utils.uuidToHex(assetDefinitionID),
      name,
      isContentPrivate,
      isContentUnique,
      contentSchemaHash
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
}

export const createUnstructuredAssetDefinition = async (assetDefinitionID: string, name: string, author: string, isContentPrivate: boolean,
  isContentUnique: boolean, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createUnstructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      assetDefinitionID: utils.uuidToHex(assetDefinitionID),
      name,
      isContentPrivate,
      isContentUnique
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
}

// Payment definition APIs

export const createDescribedPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string,
  descriptionSchemaHash: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      paymentDefinitionID,
      name,
      descriptionSchemaHash
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
};

export const createPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  const response = await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createPaymentDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: {
      paymentDefinitionID,
      name
    }
  });
  return { ...response.data, type: sync ? 'sync' : 'async' };
};

// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string, descriptionHash: string, contentHash: string, sync = false) => {

  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedAssetInstance?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { assetInstanceID, assetDefinitionID, descriptionHash, contentHash }
  });
};

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string, contentHash: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createAssetInstance?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { assetInstanceID, assetDefinitionID, contentHash }
  });
};

export const setAssetInstanceProperty = async (assetInstanceID: string, author: string, key: string, value: string, sync: boolean) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/setAssetInstanceProperty?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { assetInstanceID, key, value }
  });
};

// Payment instance APIs

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string, author: string, recipient: string, amount: number, descriptionHash: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentInstance?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { paymentInstanceID, paymentDefinitionID, recipient, amount, descriptionHash }
  });
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string, author: string, recipient: string, amount: number, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createPaymentInstance?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { paymentInstanceID, paymentDefinitionID, recipient, amount }
  });
};