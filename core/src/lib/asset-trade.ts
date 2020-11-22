import { v4 as uuidV4 } from 'uuid';
import axios from 'axios';
import Ajv from 'ajv';
import { config } from './config';
import { AssetTradeMessage, IApp2AppMessageHeader, IApp2AppMessageListener, IAssetTradePrivateAssetInstanceRequest, IAssetTradePrivateAssetInstanceResponse, IDBAssetDefinition, IDBAssetInstance, IDBMember, IDocExchangeListener, IDocExchangeTransferData } from "./interfaces";
import * as settings from './settings';
import * as utils from './utils';
import * as database from '../clients/database';
import * as app2app from '../clients/app2app';
import * as docExchange from '../clients/doc-exchange';
import { createLogger, LogLevelString } from 'bunyan';

const log = createLogger({ name: 'lib/asset-trade.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

const ajv = new Ajv();

export const assetTradeHandler = async (headers: IApp2AppMessageHeader, content: AssetTradeMessage) => {
  if (content.type === 'private-asset-instance-request') {
    processPrivateAssetInstanceRequest(headers, content);
  }
};

const processPrivateAssetInstanceRequest = async (headers: IApp2AppMessageHeader, request: IAssetTradePrivateAssetInstanceRequest) => {
  let tradeResponse: IAssetTradePrivateAssetInstanceResponse = {
    type: "private-asset-instance-response",
    tradeID: request.tradeID,
    assetInstanceID: request.assetInstanceID
  };
  const requester = await database.retrieveMemberByAddress(request.requester.address);
  try {
    if (requester === null) {
      throw new Error(`Unknown requester ${request.requester.address}`);
    }
    if (requester.assetTrailInstanceID !== request.requester.assetTrailInstanceID) {
      throw new Error(`Requester asset trail instance mismatch. Expected ${requester.assetTrailInstanceID}, ` +
        `actual ${request.requester.assetTrailInstanceID}`);
    }
    if (requester.app2appDestination !== headers.from) {
      throw new Error(`Requester App2App destination mismatch. Expected ${requester.app2appDestination}, ` +
        `actual ${headers.from}`);
    }
    const assetInstance = await database.retrieveAssetInstanceByID(request.assetInstanceID);
    if (assetInstance === null) {
      throw new Error(`Unknown asset instance ${request.assetInstanceID}`);
    }
    const author = await database.retrieveMemberByAddress(assetInstance.author);
    if (author === null) {
      throw new Error(`Unknown asset instance author`);
    }
    if (author.assetTrailInstanceID !== config.assetTrailInstanceID) {
      throw new Error(`Asset instance ${assetInstance.assetInstanceID} not authored`);
    }
    const assetDefinition = await database.retrieveAssetDefinitionByID(assetInstance.assetDefinitionID);
    if (assetDefinition === null) {
      throw new Error(`Unknown asset definition ${assetInstance.assetDefinitionID}`);
    }
    if (!assetDefinition.isContentPrivate) {
      throw new Error(`Asset instance ${assetInstance.assetInstanceID} not private`);
    }
    const authorized = await handlePrivateAssetInstanceAuthorization(assetInstance, requester, request.metadata);
    if(authorized !== true) {
      throw new Error('Access denied');
    }
    if (assetDefinition.contentSchemaHash) {
      tradeResponse.content = assetInstance.content;
    } else {
      await docExchange.transfer(config.docExchange.destination, requester.docExchangeDestination,
        utils.getUnstructuredFilePathInDocExchange(request.assetInstanceID));
      tradeResponse.filename = assetInstance.filename;
    }
  } catch (err) {
    tradeResponse.rejection = err.message;
  } finally {
    app2app.dispatchMessage(headers.from, JSON.stringify(tradeResponse), false);
  }
};

const handlePrivateAssetInstanceAuthorization = async (assetInstance: IDBAssetInstance, requester: IDBMember, metadata: object | undefined) => {
  const authorizationWebhook = settings.values[utils.settingsKeys.PRIVATE_ASSET_INSTANCE_AUTHORIZATION_WEBHOOK];
  if (authorizationWebhook === undefined) {
    throw new Error('Authorization webhook not set');
  }
  try {
    const response = await axios({
      url: authorizationWebhook,
      data: {
        assetInstance,
        requester,
        metadata
      }
    });
    return response.data.authorized;
  } catch(err) {
    log.error(`Private asset instance authorization webhook error. ${err}`);
    throw new Error('Failed to access private asset instance authorization webhook');
  }
};

export const coordinateAssetTrade = async (assetInstanceID: string, assetDefinition: IDBAssetDefinition,
  requesterAddress: string, metadata: object | undefined, authorDestination: string) => {
  const tradeID = uuidV4();
  const tradeRequest: IAssetTradePrivateAssetInstanceRequest = {
    type: 'private-asset-instance-request',
    tradeID,
    assetInstanceID,
    requester: {
      assetTrailInstanceID: config.assetTrailInstanceID,
      address: requesterAddress
    },
    metadata
  };
  const docExchangePromise = assetDefinition.contentSchema === undefined ? getDocumentExchangePromise(assetInstanceID) : Promise.resolve();
  const app2appPromise = new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      app2app.removeListener(app2appListener, false);
      reject(new Error('Asset instance author response timed out'));
    }, utils.constants.ASSET_INSTANCE_TRADE_TIMEOUT_SECONDS * 1000);
    const app2appListener: IApp2AppMessageListener = (_headers: IApp2AppMessageHeader, content: AssetTradeMessage) => {
      if (content.type === 'private-asset-instance-response' && content.tradeID === tradeID) {
        clearTimeout(timeout);
        app2app.removeListener(app2appListener, false);
        if (content.rejection) {
          reject(new Error(`Asset instance request rejected. ${content.rejection}`));
        } else {
          if (assetDefinition.contentSchema && !ajv.validate(assetDefinition.contentSchema, content.content)) {
            reject(new Error('Asset instance content does not conform to schema'));
          } else {
            database.setAssetInstancePrivateContent(content.assetInstanceID, content.content, content.filename);
            resolve();
          }
        }
      }
    };
    app2app.addListener(app2appListener, false);
    app2app.dispatchMessage(authorDestination, JSON.stringify(tradeRequest), false);
  });
  await Promise.all([app2appPromise, docExchangePromise]);
};

const getDocumentExchangePromise = (assetInstanceID: string) => {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      docExchange.removeListener(docExchangeListener);
      reject(new Error('Off-chain asset transfer timeout'));
    }, utils.constants.DOCUMENT_EXCHANGE_TRANSFER_TIMEOUT_SECONDS * 1000);
    const docExchangeListener: IDocExchangeListener = (event: IDocExchangeTransferData) => {
      if (event.document === utils.getUnstructuredFilePathInDocExchange(assetInstanceID)) {
        clearTimeout(timeout);
        docExchange.removeListener(docExchangeListener);
        resolve();
      }
    };
    docExchange.addListener(docExchangeListener);
  });
};
