import FormData from 'form-data';
import { Stream, Readable } from 'stream';
import { constants } from '../lib/utils';
import { config } from '../lib/config';
import * as utils from '../lib/utils';

export const init = async () => {
  try {
    const response = await utils.axiosWithRetry({
      url: `${config.ipfs.apiEndpoint}/api/v0/version`,
      method: 'post',
      auth: {
        username: config.appCredentials.user,
        password: config.appCredentials.password
      }
    });
    if (response.data.Version === undefined) {
      throw 'Invalid response';
    }
  } catch (err) {
    throw new Error(`IPFS Connection failed. ${err}`);
  }
};

export const downloadJSON = async <T>(hash: string): Promise<T> => {
  const response = await utils.axiosWithRetry({
    url: `${config.ipfs.gatewayEndpoint || config.ipfs.apiEndpoint}/ipfs/${hash}`,
    method: 'get',
    responseType: 'json',
    timeout: constants.IPFS_TIMEOUT_MS,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return response.data;
};

export const uploadString = (value: string): Promise<string> => {
  const readable = new Readable();
  readable.push(value);
  readable.push(null);
  return uploadStream(readable);
};

export const uploadStream = async (stream: Stream): Promise<string> => {
  const formData = new FormData();
  formData.append('document', stream);
  const response = await utils.axiosWithRetry({
    url: `${config.ipfs.apiEndpoint}/api/v0/add`,
    method: 'post',
    data: formData,
    headers: formData.getHeaders(),
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    }
  });
  return response.data.Hash;
};
