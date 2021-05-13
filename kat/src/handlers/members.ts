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

import * as database from '../clients/database';
import * as apiGateway from '../clients/api-gateway';
import * as utils from '../lib/utils';
import { IDBBlockchainData, IDBMember, IEventMemberRegistered } from '../lib/interfaces';
import RequestError from '../lib/request-handlers';
import { config } from '../lib/config';

export const handleGetMembersRequest = (query: object, skip: number, limit: number) => {
  return database.retrieveMembers(query, skip, limit);
};

export const handleGetMemberRequest = async (address: string) => {
  const member = await database.retrieveMemberByAddress(address);
  if (member === null) {
    throw new RequestError('Member not found', 404);
  }
  return member;
};

export const handleUpsertMemberRequest = async (address: string, name: string, assetTrailInstanceID: string,  app2appDestination: string, docExchangeDestination: string, sync: boolean) => {
  const timestamp = utils.getTimestamp();
  let memberDB: IDBMember = {
    address,
    name,
    assetTrailInstanceID,
    app2appDestination,
    docExchangeDestination,
    submitted: timestamp
  };
  if(config.protocol === 'ethereum') {
    const apiGatewayResponse = await apiGateway.upsertMember(address, name, app2appDestination, docExchangeDestination, sync);
    if(apiGatewayResponse.type === 'async') {
      memberDB.receipt = apiGatewayResponse.id
    }
  }
  await database.upsertMember(memberDB);
};

export const handleMemberRegisteredEvent = async ({ member, name, assetTrailInstanceID, app2appDestination, docExchangeDestination, timestamp }:
  IEventMemberRegistered, { blockNumber, transactionHash}: IDBBlockchainData) => {
  await database.upsertMember({
    address: member,
    name,
    app2appDestination,
    docExchangeDestination,
    assetTrailInstanceID,
    timestamp: Number(timestamp),
    blockNumber,
    transactionHash
  });
};
