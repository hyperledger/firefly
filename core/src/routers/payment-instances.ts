import { Router } from 'express';
import RequestError from '../lib/request-error';
import * as paymentInstancesHandler from '../handlers/payment-instances';
import { constants } from '../lib/utils';
import * as utils from '../lib/utils';

const router = Router();

router.get('/', async (req, res, next) => {
  try {
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await paymentInstancesHandler.handleGetPaymentInstancesRequest(skip, limit));
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

router.post('/', async (req, res, next) => {
  try {
    if (!req.body.paymentDefinitionID) {
      throw new RequestError('Missing payment definition ID', 400);
    }
    if (!utils.regexps.ACCOUNT.test(req.body.author)) {
      throw new RequestError('Missing or invalid payment author', 400);
    }
    if (!utils.regexps.ACCOUNT.test(req.body.recipient)) {
      throw new RequestError('Missing or invalid payment recipient', 400);
    }
    if (!(Number.isInteger(req.body.amount) && req.body.amount > 0)) {
      throw new RequestError('Missing or invalid payment amount', 400);
    }
    const sync = req.query.sync === 'true';
    const paymentInstanceID = paymentInstancesHandler.handleCreatePaymentDefinitionInstanceRequest(req.body.author, req.body.paymentDefinitionID, req.body.recipient, req.body.description, req.body.amount, sync);
    res.send({ status: 'submitted', paymentInstanceID });
  } catch (err) {
    next(err);
  }

});

export default router;
