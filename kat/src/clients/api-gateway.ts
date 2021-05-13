// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { config } from '../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse } from '../lib/interfaces';
import * as ethereumGateway from './gateway-providers/ethereum';
import * as cordaGateway from './gateway-providers/corda';

// Member APIs

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethereumGateway.upsertMember(address, name, app2appDestination, docExchangeDestination, sync);
};


// Asset definition APIs

export const createAssetDefinition = async (author: string, assetDefinitionHash: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethereumGateway.createAssetDefinition(author, assetDefinitionHash, sync);
};


// Payment definition APIs

export const createDescribedPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string,
  descriptionSchemaHash: string, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethereumGateway.createDescribedPaymentDefinition(paymentDefinitionID, name, author, descriptionSchemaHash, sync);
};

export const createPaymentDefinition = async (paymentDefinitionID: string, name: string, author: string, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  return ethereumGateway.createPaymentDefinition(paymentDefinitionID, name, author, sync);
};


// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  descriptionHash: string, contentHash: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, descriptionHash, contentHash, participants);
    case 'ethereum':
      return ethereumGateway.createDescribedAssetInstance(assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, sync);
  }
};

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string, author: string,
  contentHash: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createAssetInstance(assetInstanceID, assetDefinitionID, contentHash, participants);
    case 'ethereum':
      return ethereumGateway.createAssetInstance(assetInstanceID, assetDefinitionID, author, contentHash, sync);
  }
};

export const createAssetInstanceBatch = async (batchHash: string, author: string, participants: string[] | undefined, sync = false): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createAssetInstanceBatch(batchHash, participants);
    case 'ethereum':
      return ethereumGateway.createAssetInstanceBatch(batchHash, author, sync);
  }
}

export const setAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, author: string, key: string, value: string,
  participants: string[] | undefined, sync: boolean): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.setAssetInstanceProperty(assetDefinitionID, assetInstanceID, key, value, participants);
    case 'ethereum':
      return ethereumGateway.setAssetInstanceProperty(assetDefinitionID, assetInstanceID, author, key, value, sync);
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
      return ethereumGateway.createDescribedPaymentInstance(paymentInstanceID, paymentDefinitionID, author, recipient, amount, descriptionHash, sync);
  }
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  author: string, recipient: string, amount: number, participants: string[] | undefined, sync: boolean):
  Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  switch (config.protocol) {
    case 'corda':
      return cordaGateway.createPaymentInstance(paymentInstanceID, paymentDefinitionID, recipient, amount, participants);
    case 'ethereum':
      return ethereumGateway.createPaymentInstance(paymentInstanceID, paymentDefinitionID, author, recipient, amount, sync);
  }
};
