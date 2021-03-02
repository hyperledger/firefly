import { Router } from 'express';
import { v4 as uuidV4 } from 'uuid';
import RequestError from '../lib/request-error';
import * as assetDefinitionsHandler from '../handlers/asset-definitions';
import { constants } from '../lib/utils';
import * as utils from '../lib/utils';
import { config } from '../lib/config';

const router = Router();

router.get('/', async (req, res, next) => {
  try {
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await assetDefinitionsHandler.handleGetAssetDefinitionsRequest({}, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:assetDefinitionID', async (req, res, next) => {
  try {
    res.send(await assetDefinitionsHandler.handleGetAssetDefinitionRequest(req.params.assetDefinitionID));
  } catch (err) {
    next(err);
  }
});

router.post('/search', async (req, res, next) => {
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
      await assetDefinitionsHandler.handleCountAssetDefinitionsRequest(req.body.query) :
      await assetDefinitionsHandler.handleGetAssetDefinitionsRequest(req.body.query, skip, limit)
    );
  } catch (err) {
    next(err);
  }
});

router.post('/', async (req, res, next) => {
  try {
    if (!req.body.name || req.body.name === '') {
      throw new RequestError('Missing or invalid asset definition name', 400);
    }
    if (!utils.isAuthorValid(req.body.author, config.protocol)) {
      throw new RequestError('Missing or invalid asset definition author', 400);
    }
    if (typeof req.body.isContentPrivate !== 'boolean') {
      throw new RequestError('Missing asset definition content privacy', 400);
    }
    if (typeof req.body.isContentUnique !== 'boolean') {
      throw new RequestError('Missing asset definition content uniqueness', 400);
    }
    let assetDefinitionID;
    switch (config.protocol) {
      case 'corda':
        if (!req.body.assetDefinitionID) {
          throw new RequestError('Missing asset definition id', 400);
        }
        assetDefinitionID = req.body.assetDefinitionID;
        break;
      case 'ethereum':
        assetDefinitionID = uuidV4();
        break;
    }
    const sync = req.query.sync === 'true';
    await assetDefinitionsHandler.handleCreateAssetDefinitionRequest(assetDefinitionID, req.body.name, req.body.isContentPrivate,
      req.body.isContentUnique, req.body.author, req.body.descriptionSchema, req.body.contentSchema, req.body.indexes, sync);
    res.send({ status: sync ? 'success' : 'submitted', assetDefinitionID });
  } catch (err) {
    next(err);
  }
});

export default router;
