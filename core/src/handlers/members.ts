import * as database from '../clients/database';
import * as apiGateway from '../clients/api-gateway';
import * as utils from '../lib/utils';
import { IDBBlockchainData, IEventMemberRegistered } from '../lib/interfaces';
import RequestError from '../lib/request-error';
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

export const handleUpsertMemberRequest = async (address: string, name: string, participants: string[] | undefined, sync: boolean) => {
  const timestamp = utils.getTimestamp();
  const apiGatewayResponse = await apiGateway.upsertMember(address, name, config.app2app.destinations.kat, config.docExchange.destination, participants, sync);
  const receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  await database.upsertMember({
    address,
    name,
    assetTrailInstanceID: config.assetTrailInstanceID,
    app2appDestination: config.app2app.destinations.kat,
    docExchangeDestination: config.docExchange.destination,
    submitted: timestamp,
    receipt,
    participants
  });
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
