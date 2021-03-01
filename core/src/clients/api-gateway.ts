import axios from 'axios';
import { config } from '../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse } from '../lib/interfaces';
import * as utils from '../lib/utils';

// Member APIs

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'ethereum':
      return upsertMemberEthereum(address, name, app2appDestination, docExchangeDestination, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const upsertMemberEthereum = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

// Asset definition APIs

export const createAssetDefinition = async (author: string, assetDefinitionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'ethereum':
      return createAssetDefinitionEthereum(author, assetDefinitionHash, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const createAssetDefinitionEthereum = async (author: string, assetDefinitionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

// Payment definition APIs
export const createDescribedPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string,
  descriptionSchemaHash: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'ethereum':
      return createDescribedPaymentDefinitionEthereum(paymentDefinitionID, name, author, descriptionSchemaHash, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const createDescribedPaymentDefinitionEthereum = async (paymentDefinitionID: string, name: string, author: string,
  descriptionSchemaHash: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

export const createPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'ethereum':
      return createPaymentDefinitionEthereum(paymentDefinitionID, name, author, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const createPaymentDefinitionEthereum = async (paymentDefinitionID: string, name: string, author: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  descriptionHash: string, contentHash: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return createDescribedAssetInstanceCorda(assetInstanceID, assetDefinitionID, descriptionHash, contentHash, participants);
    case 'ethereum':
      return createDescribedAssetInstanceEthereum(assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};


const createDescribedAssetInstanceEthereum = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  descriptionHash: string, contentHash: string, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

const createDescribedAssetInstanceCorda = async (assetInstanceID: string, assetDefinitionID: string,
  descriptionHash: string, contentHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedAssetInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        descriptionHash,
        contentHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  contentHash: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return createAssetInstanceCorda(assetInstanceID, assetDefinitionID, contentHash, participants);
    case 'ethereum':
      return createAssetInstanceEthereum(assetInstanceID, assetDefinitionID, author, contentHash, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const createAssetInstanceEthereum = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  contentHash: string, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

const createAssetInstanceCorda = async (assetInstanceID: string, assetDefinitionID: string,
  contentHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetInstance`,
      auth: {
        username: config.apiGateway.auth ? config.apiGateway.auth.user : config.appCredentials.user,
        password: config.apiGateway.auth ? config.apiGateway.auth.password : config.appCredentials.password
      },
      data: {
        assetInstanceID: utils.uuidToHex(assetInstanceID),
        assetDefinitionID: utils.uuidToHex(assetDefinitionID),
        contentHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

export const createAssetInstanceBatch = async (batchHash: string, author: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return createAssetInstanceBatchCorda(batchHash, participants);
    case 'ethereum':
      return createAssetInstanceBatchEthereum(batchHash, author, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
}

const createAssetInstanceBatchEthereum = async (batchHash: string, author: string, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

const createAssetInstanceBatchCorda = async (batchHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetInstanceBatch`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        batchHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

export const setAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string,
  participants: string[] | undefined, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return setAssetInstancePropertyCorda(assetDefinitionID, assetInstanceID, key, value, participants);
    case 'ethereum':
      return setAssetInstancePropertyEthereum(assetDefinitionID, assetInstanceID, author, key, value, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const setAssetInstancePropertyEthereum = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string,
  sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/setAssetInstanceProperty?kld-from=${author}&kld-sync=${sync}`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetDefinitionID,
        assetInstanceID,
        key,
        value
      }
    });
    return { ...response.data, type: sync ? 'sync' : 'async' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

const setAssetInstancePropertyCorda = async (assetDefinitionID: string, assetInstanceID: string, key: string, value: string,
  participants: string[] | undefined | undefined): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/setAssetInstanceProperty`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetDefinitionID,
        assetInstanceID,
        key,
        value,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

// Payment instance APIs

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, descriptionHash: string, participants: string[] | undefined, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return createDescribedPaymentInstanceCorda(paymentInstanceID, paymentDefinitionID, recipient, amount, descriptionHash, participants);
    case 'ethereum':
      return createDescribedPaymentInstanceEthereum(paymentInstanceID, paymentDefinitionID, author, recipient, amount, descriptionHash, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const createDescribedPaymentInstanceEthereum = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, descriptionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

const createDescribedPaymentInstanceCorda = async (paymentInstanceID: string, paymentDefinitionID: string, recipient: string, amount: number, descriptionHash: string, participants: string[] | undefined):
  Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentInstanceID: utils.uuidToHex(paymentInstanceID),
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        recipient,
        amount,
        descriptionHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, participants: string[] | undefined, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return createPaymentInstanceCorda(paymentInstanceID, paymentDefinitionID, recipient, amount, participants);
    case 'ethereum':
      return createPaymentInstanceEthereum(paymentInstanceID, paymentDefinitionID, author, recipient, amount, sync);
    default:
      throw new Error("Unsupported protocol.");
  }
};

const createPaymentInstanceEthereum = async (paymentInstanceID: string, paymentDefinitionID: string, author: string,
  recipient: string, amount: number, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
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
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};

const createPaymentInstanceCorda = async (paymentInstanceID: string, paymentDefinitionID: string,
  recipient: string, amount: number, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createPaymentInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentInstanceID: utils.uuidToHex(paymentInstanceID),
        paymentDefinitionID: utils.uuidToHex(paymentDefinitionID),
        recipient,
        amount,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    throw new Error(err.response?.data?.error ?? err.response.data.message ?? err.toString());
  }
};
