import { Router } from 'express';
import RequestError from '../lib/request-error';
import * as paymentDefinitionsHandler from '../handlers/payment-definitions';
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
    res.send(await paymentDefinitionsHandler.handleGetPaymentDefinitionsRequest({}, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:paymentDefinitionID', async (req, res, next) => {
  try {
    res.send(await paymentDefinitionsHandler.handleGetPaymentDefinitionRequest(req.params.paymentDefinitionID));
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
      await paymentDefinitionsHandler.handleCountPaymentDefinitionsRequest(req.body.query) :
      await paymentDefinitionsHandler.handleGetPaymentDefinitionsRequest(req.body.query, skip, limit)
    );
  } catch (err) {
    next(err);
  }
});

router.post('/', async (req, res, next) => {
  try {
    if (!req.body.name || req.body.name === '') {
      throw new RequestError('Missing or invalid payment definition name', 400);
    }
    if (!utils.isAuthorValid(req.body.author, config.protocol)) {
      throw new RequestError('Missing or invalid payment definition author', 400);
    }
    const sync = req.query.sync === 'true';
    const paymentDefinitionID = await paymentDefinitionsHandler.handleCreatePaymentDefinitionRequest(req.body.name,
      req.body.author, req.body.descriptionSchema, sync);
    res.send({ status: sync? 'success' : 'submitted', paymentDefinitionID });
  } catch (err) {
    next(err);
  }
});

export default router;
