import * as database from '../clients/database';
import * as apiGateway from '../clients/api-gateway';
import * as utils from '../lib/utils';
import { IDBBlockchainData, IEventMemberRegistered } from '../lib/interfaces';
import RequestError from '../lib/request-error';
import { config } from '../lib/config';

export const handleGetMembersRequest = (skip: number, limit: number, owned: boolean) => {
  return database.retrieveMembers(skip, limit, owned);
};

export const handleGetMemberRequest = async (address: string) => {
  const member = await database.retrieveMemberByAddress(address);
  if (member === null) {
    throw new RequestError('Member not found', 404);
  }
  return member;
};

export const handleUpsertMemberRequest = async (address: string, name: string, sync: boolean) => {
  const response = await apiGateway.upsertMember(address, name, config.app2app.destination, config.docExchange.destination, sync);
  let receipt = response.type === 'async'? response.id : undefined;
  await database.upsertMemberFromRequest(address, name, config.assetTrailInstanceID, config.app2app.destination,
    config.docExchange.destination, receipt, utils.getTimestamp());
};

export const handleMemberRegisteredEvent = async ({ member, name, app2appDestination, docExchangeDestination, timestamp }:
  IEventMemberRegistered, blockchainData: IDBBlockchainData) => {
  await database.upsertMemberFromEvent(member, name, config.assetTrailInstanceID, app2appDestination, docExchangeDestination,
    Number(timestamp), blockchainData);
};
