import { Router, Request } from 'express';
import RequestError from '../lib/request-error';
import * as assetInstancesHandler from '../handlers/asset-instances';
import { constants, requestKeys, streamToString } from '../lib/utils';
import Busboy from 'busboy';
import * as utils from '../lib/utils';
import { IRequestMultiPartContent } from '../lib/interfaces';

const router = Router();

const MongoQS = require('mongo-querystring');
const qs = new MongoQS({
  blacklist: { skip: true, limit: true }
});

router.get('/', async (req, res, next) => {
  try {
    const query = qs.parse(req.query);
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await assetInstancesHandler.handleGetAssetInstancesRequest(query, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:assetInstanceID', async (req, res, next) => {
  try {
    const content = req.query.content === 'true';
    res.send(await assetInstancesHandler.handleGetAssetInstanceRequest(req.params.assetInstanceID, content));
  } catch (err) {
    next(err);
  }
});

router.post('/', async (req, res, next) => {
  try {
    let assetInstanceID: string;
    const sync = req.query.sync === 'true';
    if (req.headers["content-type"]?.startsWith('multipart/form-data')) {
      let description: Object | undefined;
      const formData = await extractDataFromMultipartForm(req);
      if (!formData.assetDefinitionID) {
        throw new RequestError('Missing asset definition ID', 400);
      }
      if (formData.description !== undefined) {
        try {
          description = JSON.parse(await formData.description);
        } catch (err) {
          throw new RequestError(`Invalid description. ${err}`, 400);
        }
      }
      if (!formData.author || !utils.regexps.ACCOUNT.test(formData.author)) {
        throw new RequestError('Missing or invalid asset instance author', 400);
      }
      assetInstanceID = await assetInstancesHandler.handleCreateUnstructuredAssetInstanceRequest(formData.author, formData.assetDefinitionID, description, formData.contentStream, formData.contentFileName, sync);
    } else {
      if (!req.body.assetDefinitionID) {
        throw new RequestError('Missing asset definition ID', 400);
      }
      if (!utils.regexps.ACCOUNT.test(req.body.author)) {
        throw new RequestError('Missing or invalid asset instance author', 400);
      }
      if (!(typeof req.body.content === 'object' && req.body.content !== null)) {
        throw new RequestError('Missing or invalid asset content', 400);
      }
      assetInstanceID = await assetInstancesHandler.handleCreateStructuredAssetInstanceRequest(req.body.author, req.body.assetDefinitionID, req.body.description, req.body.content, sync);
    }
    res.send({ status: sync ? 'success' : 'submitted', assetInstanceID });
  } catch (err) {
    next(err);
  }
});

router.put('/:assetInstanceID', async (req, res, next) => {
  try {
    if (!req.body.key) {
      throw new RequestError('Missing asset property key', 400);
    }
    if (!req.body.value) {
      throw new RequestError('Missing asset property value', 400);
    }
    if (!utils.regexps.ACCOUNT.test(req.body.author)) {
      throw new RequestError('Missing or invalid asset property author', 400);
    }
    const sync = req.query.sync === 'true';
    await assetInstancesHandler.handleSetAssetInstancePropertyRequest(req.params.assetInstanceID, req.body.author, req.body.key, req.body.value, sync);
    res.send({ status: sync ? 'success' : 'submitted' });
  } catch (err) {
    next(err);
  }
});

router.patch('/:assetInstanceID', async (req, res, next) => {
  try {
    if (!utils.regexps.ACCOUNT.test(req.body.requester)) {
      throw new RequestError(`Missing requester`);
    }
    await assetInstancesHandler.handleAssetInstanceTradeRequest(req.body.requester, req.params.assetInstanceID, req.body.metadata);
    res.send({ status: 'success' });
  } catch (err) {
    next(err);
  }
});

const extractDataFromMultipartForm = (req: Request): Promise<IRequestMultiPartContent> => {
  return new Promise(async (resolve, reject) => {
    let author: string | undefined;
    let assetDefinitionID: string | undefined;
    let description: Promise<string> | undefined;
    req.pipe(new Busboy({ headers: req.headers })
      .on('field', (fieldname, value) => {
        switch (fieldname) {
          case requestKeys.ASSET_AUTHOR: author = value; break;
          case requestKeys.ASSET_DEFINITION_ID: assetDefinitionID = value; break;
        }
      }).on('file', (fieldname, readableStream, fileName) => {
        switch (fieldname) {
          case requestKeys.ASSET_DESCRIPTION: description = streamToString(readableStream); break;
          case requestKeys.ASSET_CONTENT: resolve({ author, assetDefinitionID, description, contentStream: readableStream, contentFileName: fileName }); break;
          default: readableStream.resume();
        }
      })).on('finish', () => {
        reject(new RequestError('Missing content', 400));
      });
  });
}

export default router;
