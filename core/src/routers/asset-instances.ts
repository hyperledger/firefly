import { Router, Request } from 'express';
import RequestError from '../lib/request-handlers';
import * as assetInstancesHandler from '../handlers/asset-instances';
import { constants, requestKeys, streamToString } from '../lib/utils';
import Busboy from 'busboy';
import * as utils from '../lib/utils';
import { IRequestMultiPartContent } from '../lib/interfaces';
import { config } from '../lib/config';

const router = Router();

router.get('/:assetDefinitionID', async (req, res, next) => {
  try {
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await assetInstancesHandler.handleGetAssetInstancesRequest(req.params.assetDefinitionID, {}, {}, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:assetDefinitionID/:assetInstanceID', async (req, res, next) => {
  try {
    const content = req.query.content === 'true';
    res.send(await assetInstancesHandler.handleGetAssetInstanceRequest(req.params.assetDefinitionID, req.params.assetInstanceID, content));
  } catch (err) {
    next(err);
  }
});

router.post('/search/:assetDefinitionID', async (req, res, next) => {
  try {
    const skip = Number(req.body.skip || 0);
    const limit = Number(req.body.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (req.body.count !== true && (isNaN(skip) || isNaN(limit))) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    if (!req.body.query) {
      throw new RequestError('Missing search query', 400);
    }
    res.send(req.body.count === true ?
      await assetInstancesHandler.handleCountAssetInstancesRequest(req.params.assetDefinitionID, req.body.query) :
      await assetInstancesHandler.handleGetAssetInstancesRequest(req.params.assetDefinitionID, req.body.query, req.body.sort || {}, skip, limit)
    );
  } catch (err) {
    next(err);
  }
});

router.post('/:assetDefinitionID', async (req, res, next) => {
  try {
    let assetInstanceID: string;
    const sync = req.query.sync === 'true';

    if (req.headers["content-type"]?.startsWith('multipart/form-data')) {
      let description: Object | undefined;
      const formData = await extractDataFromMultipartForm(req);
      if (formData.description !== undefined) {
        try {
          description = JSON.parse(await formData.description);
        } catch (err) {
          throw new RequestError(`Invalid description. ${err}`, 400);
        }
      }
      if (!formData.author ||!utils.isAuthorValid(formData.author, config.protocol)) {
        throw new RequestError('Missing or invalid asset instance author', 400);
      }
      assetInstanceID = await assetInstancesHandler.handleCreateUnstructuredAssetInstanceRequest(formData.author, req.params.assetDefinitionID, description, formData.contentStream, formData.contentFileName, formData.isContentPrivate, req.body.participants, sync);
    } else {
      if (!utils.isAuthorValid(req.body.author, config.protocol)) {
        throw new RequestError('Missing or invalid asset instance author', 400);
      }
      if (!(typeof req.body.content === 'object' && req.body.content !== null)) {
        throw new RequestError('Missing or invalid asset content', 400);
      }
      if(req.body.isContentPrivate !== undefined && typeof req.body.isContentPrivate !== 'boolean') {
        throw new RequestError('Invalid isContentPrivate', 400);
      }
      assetInstanceID = await assetInstancesHandler.handleCreateStructuredAssetInstanceRequest(req.body.author, req.params.assetDefinitionID, req.body.description, req.body.content, req.body.isContentPrivate, req.body.participants, sync);
    }
    res.send({ status: sync ? 'success' : 'submitted', assetInstanceID });
  } catch (err) {
    next(err);
  }
});

router.put('/:assetDefinitionID/:assetInstanceID', async (req, res, next) => {
  try {
    switch (req.body.action) {
      case 'set-property':
        if (!req.body.key) {
          throw new RequestError('Missing asset property key', 400);
        }
        if (!req.body.value) {
          throw new RequestError('Missing asset property value', 400);
        }
        if (!utils.isAuthorValid(req.body.author, config.protocol)) {
          throw new RequestError('Missing or invalid asset property author', 400);
        }
        const sync = req.query.sync === 'true';
        await assetInstancesHandler.handleSetAssetInstancePropertyRequest(req.params.assetDefinitionID, req.params.assetInstanceID, req.body.author, req.body.key, req.body.value, sync);
        res.send({ status: sync ? 'success' : 'submitted' });
        break;
      case 'push':
        if (!req.body.recipientAddress) {
          throw new RequestError('Missing recipient address', 400);
        }
        await assetInstancesHandler.handlePushPrivateAssetInstanceRequest(req.params.assetDefinitionID, req.params.assetInstanceID, req.body.recipientAddress);
        res.send({ status: 'success' });
        break;
      default:
        throw new RequestError('Missing or invalid action');
    }
  } catch (err) {
    next(err);
  }
});

router.patch('/:assetDefinitionID/:assetInstanceID', async (req, res, next) => {
  try {
    if (!utils.isAuthorValid(req.body.requester, config.protocol)) {
      throw new RequestError(`Missing requester`);
    }
    await assetInstancesHandler.handleAssetInstanceTradeRequest(req.params.assetDefinitionID, req.body.requester, req.params.assetInstanceID, req.body.metadata);
    res.send({ status: 'success' });
  } catch (err) {
    next(err);
  }
});

router.post('/:assetDefinitionID/:assetInstanceID/push', async (req, res, next) => {
  try {
    if (!req.body.recipientAddress) {
      throw new RequestError('Missing recipient address', 400);
    }
    await assetInstancesHandler.handlePushPrivateAssetInstanceRequest(req.params.assetDefinitionID, req.params.assetInstanceID, req.body.recipientAddress);
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
    let isContentPrivate: boolean | undefined = undefined;
    req.pipe(new Busboy({ headers: req.headers })
      .on('field', (fieldname, value) => {
        switch (fieldname) {
          case requestKeys.ASSET_AUTHOR: author = value; break;
          case requestKeys.ASSET_DEFINITION_ID: assetDefinitionID = value; break;
          case requestKeys.ASSET_IS_CONTENT_PRIVATE: isContentPrivate = value === 'true'; break;
        }
      }).on('file', (fieldname, readableStream, fileName) => {
        switch (fieldname) {
          case requestKeys.ASSET_DESCRIPTION: description = streamToString(readableStream); break;
          case requestKeys.ASSET_CONTENT: resolve({ author, assetDefinitionID, description, contentStream: readableStream, contentFileName: fileName, isContentPrivate }); break;
          default: readableStream.resume();
        }
      })).on('finish', () => {
        reject(new RequestError('Missing content', 400));
      });
  });
};

export default router;
