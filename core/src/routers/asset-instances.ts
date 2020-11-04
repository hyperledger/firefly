import { Router, Request } from 'express';
import RequestError from '../lib/request-error';
import * as assetInstancesHandler from '../handlers/asset-instances';
import { constants, requestKeys, streamToString } from '../lib/utils';
import Ajv from 'ajv';
import Busboy from 'busboy';
import { IRequestMultiPartContent } from '../lib/interfaces';

const ajv = new Ajv();

const router = Router();

router.get('/', async (req, res, next) => {
  try {
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await assetInstancesHandler.handleGetAssetInstancesRequest(skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:assetInstanceID', async (req, res, next) => {
  try {
    const assetInstanceID = Number(req.params.assetInstanceID);
    if (isNaN(assetInstanceID)) {
      throw new RequestError('Invalid asset instance ID', 400);
    }
    res.send(await assetInstancesHandler.handleGetAssetInstanceRequest(assetInstanceID));
  } catch (err) {
    next(err);
  }
});

router.post('/', async (req, res, next) => {
  try {
    if (req.headers["content-type"]?.startsWith('multipart/form-data')) {
      let description: Object | undefined;
      const formData = await extractDataFromMultipartForm(req);
      if (formData.assetType === undefined || isNaN(formData.assetType)) {
        throw new RequestError('Missing or invalid asset definition ID', 400);
      }
      if (formData.description !== undefined) {
        description = JSON.parse(await formData.description);
      }
      if (!formData.author) {
        throw new RequestError('Missing author', 400);
      }
      await assetInstancesHandler.handleCreateAssetInstanceMultiPartRequest(formData.author, formData.assetType, description, formData.contentStream);

      res.send('ok')
    } else {
      if (!(typeof req.body.assetDefinitionID === 'number')) {
        throw new RequestError('Missing or invalid asset definition ID', 400);
      }
      if(!req.body.author) {
        throw new RequestError('Missing asset author', 400);
      }
      if (!(typeof req.body.content === 'object' && req.body.content !== null)) {
        throw new RequestError('Missing or invalid asset content', 400);
      }
      await assetInstancesHandler.handleCreateAssetInstanceRequest(req.body.author, req.body.assetDefinitionID, req.body.description, req.body.content);
    }
  } catch (err) {
    next(err);
  }
});

const extractDataFromMultipartForm = (req: Request): Promise<IRequestMultiPartContent> => {
  return new Promise(async (resolve, reject) => {
    let author: string | undefined;
    let assetType: number | undefined;
    let description: Promise<string> | undefined;
    req.pipe(new Busboy({ headers: req.headers })
      .on('field', (fieldname, value) => {
        switch (fieldname) {
          case requestKeys.ASSET_AUTHOR: author = value; break;
          case requestKeys.ASSET_DEFINITION_ID: assetType = Number(value); break;
        }
      }).on('file', (fieldname, readableStream, fileName) => {
        switch (fieldname) {
          case requestKeys.ASSET_DESCRIPTION: description = streamToString(readableStream); break;
          case requestKeys.ASSET_CONTENT: resolve({ author, assetType, description, contentStream: readableStream, contentFileName: fileName }); break;
          default: readableStream.resume();
        }
      })).on('finish', () => {
        reject(new RequestError('Missing content', 400));
      });
  });
}

export default router;
