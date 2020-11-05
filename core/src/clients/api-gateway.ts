import axios from 'axios';
import { config } from '../lib/config';
import { IStatus } from '../lib/interfaces';

export const init = async () => {
  try {
    await getStatus();
  } catch (err) {
    throw new Error(`Failed to access API Gateway. ${err}`);
  }
};

export const getStatus = async (): Promise<IStatus> => {
  const response = await axios({
    url: `${config.apiGateway.apiEndpoint}/getStatus`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return {
    totalAssetDefinitions: Number(response.data.totalAssetDefinitions),
    totalPaymentDefinitions: Number(response.data.totalPaymentDefinitionsc)
  };
};

// Member APIs

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/registerMember?kld-from=${address}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, app2appDestination, docExchangeDestination }
  });
};

// Asset definition APIs

export const createDescribedStructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, descriptionSchemaHash: string, contentSchemaHash: string) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedStructuredAssetDefinition?kld-from=${author}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate, descriptionSchemaHash, contentSchemaHash }
  });
}

export const createDescribedUnstructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, descriptionSchemaHash: string) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedUnstructuredAssetDefinition?kld-from=${author}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate, descriptionSchemaHash }
  });
}

export const createStructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, contentSchemaHash: string) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createStructuredAssetDefinition?kld-from=${author}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate, contentSchemaHash }
  });
}

export const createUnstructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createUnstructuredAssetDefinition?kld-from=${author}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate }
  });
}

// Payment definition APIs

export const createDescribedPaymentDefinition = async (name: string, author: string, amount: number, descriptionSchemaHash: string) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentDefinition?kld-from=${author}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, descriptionSchemaHash, amount }
  });
};

export const createPaymentDefinition = async (name: string, author: string, amount: number) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createPaymentDefinition?kld-from=${author}&kld-sync=true`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, amount }
  });
};

// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: number, author: string, descriptionHash: string, contentHash: string, sync = false) => {
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

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: number, author: string, contentHash: string, sync=false) => {
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

// Payment instance APIs

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: number, author: string, recipient: string, descriptionHash: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentInstance?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { paymentInstanceID, paymentDefinitionID, recipient, descriptionHash }
  });
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: number, author: string, recipient: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createPaymentInstance?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { paymentInstanceID, paymentDefinitionID, recipient }
  });
};