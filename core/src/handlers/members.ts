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
