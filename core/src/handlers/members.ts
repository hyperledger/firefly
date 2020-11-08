import * as database from '../clients/database';
import * as apiGateway from '../clients/api-gateway';
import * as utils from '../lib/utils';
import { IEventMemberRegistered } from '../lib/interfaces';
import RequestError from '../lib/request-error';

export const handleGetMembersRequest = (skip: number, limit: number, owned: boolean) => {
  return database.retrieveMembers(skip, limit, owned);
};

export const handleGetMemberRequest = async (address: string) => {
  const member = await database.retrieveMemberByAddress(address);
  if(member === null) {
    throw new RequestError('Member not found', 404);
  }
  return member;
};

export const handleUpsertMemberRequest = async (address: string, name: string,
  app2appDestination: string, docExchangeDestination: string, sync: boolean) => {
  await database.upsertMember(address, name, app2appDestination, docExchangeDestination, utils.getTimestamp(), false, true);
  await apiGateway.upsertMember(address, name, app2appDestination, docExchangeDestination, sync);
};

export const handleMemberRegisteredEvent = async ({ member, name, app2appDestination, docExchangeDestination, timestamp }: IEventMemberRegistered) => {
  const dbMember = await database.retrieveMemberByAddress(member);
  const memberOwned = dbMember?.owned ?? false;
  await database.upsertMember(member, name, app2appDestination, docExchangeDestination, Number(timestamp), true, memberOwned);
};
