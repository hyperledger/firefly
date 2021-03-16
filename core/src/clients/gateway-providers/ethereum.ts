import axios from 'axios';
import { config } from '../../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponseEthereum } from '../../lib/interfaces';
import * as utils from '../../lib/utils';
const log = utils.getLogger('gateway-providers/ethereum.ts');

// Member APIs

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/registerMember?kld-from=${address}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        name,
        assetTrailInstanceID: config.assetTrailInstanceID,
        app2appDestination,
        docExchangeDestination
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to register new member ${name}`, err);
  }
};


// Asset definition APIs

export const createAssetDefinition = async (author: string, assetDefinitionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetDefinition?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetDefinitionHash
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create asset definition ${assetDefinitionHash}`, err);
  }
};


// Payment definition APIs

export const createDescribedPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string,
  descriptionSchemaHash: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentDefinition?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        name,
        descriptionSchemaHash
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create described payment definition ${paymentDefinitionID}`, err);
  }
};

export const createPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createPaymentDefinition?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        name
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create payment definition ${paymentDefinitionID}`, err);
  }
};


// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  descriptionHash: string, contentHash: string, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedAssetInstance?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        descriptionHash,
        contentHash
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create described asset instance ${assetInstanceID}`, err);
  }
};

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  contentHash: string, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetInstance?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        contentHash
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create asset instance ${assetInstanceID}`, err);
  }
};

export const createAssetInstanceBatch = async (batchHash: string, author: string, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetInstanceBatch?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        batchHash,
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create asset instance batch ${batchHash}`, err);
  }
};

export const setAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string,
  sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/setAssetInstanceProperty?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        key,
        value
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to set asset instance property ${key} (instance=${assetInstanceID})`, err);
  }
};


// Payment instance APIs

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, descriptionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentInstance?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentInstanceID: utils.uuidToHex(paymentInstanceID),
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        recipient,
        amount,
        descriptionHash
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create described payment instance ${paymentInstanceID}`, err);
  }
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string, author: string,
  recipient: string, amount: number, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createPaymentInstance?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentInstanceID: utils.uuidToHex(paymentInstanceID),
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        recipient,
        amount
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    return handleError(`Failed to create payment instance ${paymentInstanceID}`, err);
  }
};

function handleError(msg: string, err: any): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponseEthereum> {
  const errMsg = err.response?.data?.error ?? err.response.data.message ?? err.toString();
  log.error(`${msg}. ${errMsg}`);
  throw new Error(msg);
}