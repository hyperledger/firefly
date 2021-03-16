import { Router } from 'express';
import RequestError from '../lib/request-handlers';
import * as paymentInstancesHandler from '../handlers/payment-instances';
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
    res.send(await paymentInstancesHandler.handleGetPaymentInstancesRequest({}, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:assetInstanceID', async (req, res, next) => {
  try {
    res.send(await paymentInstancesHandler.handleGetPaymentInstanceRequest(req.params.assetInstanceID));
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
      await paymentInstancesHandler.handleCountPaymentInstancesRequest(req.body.query) :
      await paymentInstancesHandler.handleGetPaymentInstancesRequest(req.body.query, skip, limit)
    );
  } catch (err) {
    next(err);
  }
});

router.post('/', async (req, res, next) => {
  try {
    if (!req.body.paymentDefinitionID) {
      throw new RequestError('Missing payment definition ID', 400);
    }
    if (!utils.isAuthorValid(req.body.author, config.protocol)) {
      throw new RequestError('Missing or invalid payment author', 400);
    }
    if (!utils.isAuthorValid(req.body.recipient, config.protocol)) {
      throw new RequestError('Missing or invalid payment recipient', 400);
    }
    if (req.body.author === req.body.recipient) {
      throw new RequestError('Author and recipient cannot be the same', 400);
    }
    if (!(Number.isInteger(req.body.amount) && req.body.amount > 0)) {
      throw new RequestError('Missing or invalid payment amount', 400);
    }
    const sync = req.query.sync === 'true';
    const paymentInstanceID = await paymentInstancesHandler.handleCreatePaymentInstanceRequest(req.body.author, req.body.paymentDefinitionID, req.body.recipient, req.body.description, req.body.amount, req.body.participants, sync);
    res.send({ status: sync? 'success' : 'submitted', paymentInstanceID });
  } catch (err) {
    next(err);
  }

});

export default router;
