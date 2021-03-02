import axios from 'axios';
import { config } from '../../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse } from '../../lib/interfaces';
import * as utils from '../../lib/utils';

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string,
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

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string,
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

export const createAssetInstanceBatch = async (batchHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
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

export const setAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, key: string, value: string,
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

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string, recipient: string, amount: number, descriptionHash: string, participants: string[] | undefined):
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
