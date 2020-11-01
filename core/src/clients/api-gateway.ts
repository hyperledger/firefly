import axios from 'axios';
import { config } from '../lib/config';
import { IStatus } from '../lib/interfaces';

export const init = () => {
  // TODO: sanity test
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
    totalAssetInstances: Number(response.data.totalAssetInstances),
    totalPaymentDefinitionsc: Number(response.data.totalPaymentDefinitionsc),
    totalPaymentInstances: Number(response.data.totalPaymentInstances)
  };
};

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/registerMember?kld-from=${address}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, app2appDestination, docExchangeDestination }
  });
};

export const createDescribedStructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, descriptionSchemaHash: string, contentSchemaHash: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedStructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate, descriptionSchemaHash, contentSchemaHash }
  });
}

export const createDescribedUnstructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, descriptionSchemaHash: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createDescribedUnstructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate, descriptionSchemaHash }
  });
}

export const createStructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, contentSchemaHash: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createStructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate, contentSchemaHash }
  });
}

export const createUnstructuredAssetDefinition = async (name: string, author: string, isContentPrivate: boolean, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/createUnstructuredAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, isContentPrivate }
  });
}