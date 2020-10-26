import * as database from '../clients/database';
import * as apiGateway from '../clients/api-gateway';
import * as utils from '../lib/utils';
import { IMemberRegisteredEvent } from '../lib/interfaces';

export const handleGetMembersRequest = (skip: number, limit: number) => {
  return database.retrieveMembers(skip, limit);
};

export const handleGetMemberRequest = (address: string) => {
  return database.retrieveMember(address);
};

export const handleUpsertMemberRequest = async (address: string, name: string,
  app2appDestination: string, docExchangeDestination: string, sync: boolean) => {
  await apiGateway.upsertMember(address, name, app2appDestination, docExchangeDestination, sync);
  if(!sync) {
    await database.upsertMember(address, name, app2appDestination, docExchangeDestination, utils.getTimestamp(), false, true);
  }
};

export const handleMemberRegisteredEvent = async ({ member, name, app2appDestination, docExchangeDestination, timestamp }: IMemberRegisteredEvent) => {
  let owned = false;
  const currentValue = await database.retrieveMember(member);
  if(currentValue !== null) {
    owned = currentValue.owned;
  }
  await database.upsertMember(member, name, app2appDestination, docExchangeDestination, Number(timestamp), true, owned);
};
