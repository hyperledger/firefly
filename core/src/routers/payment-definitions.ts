import { Router } from 'express';
import RequestError from '../lib/request-error';
import * as paymentDefinitionsHandler from '../handlers/payment-definitions';
import { constants } from '../lib/utils';
import * as utils from '../lib/utils';
import Ajv from 'ajv';

const ajv = new Ajv();

const router = Router();

router.get('/', async (req, res, next) => {
  try {
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await paymentDefinitionsHandler.handleGetPaymentDefinitionsRequest(skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:paymentDefinitionID', async (req, res, next) => {
  try {
    const paymentDefinitionID = Number(req.params.paymentDefinitionID);
    if (isNaN(paymentDefinitionID)) {
      throw new RequestError('Invalid payment definition ID', 400);
    }
    res.send(await paymentDefinitionsHandler.handleGetPaymentDefinitionRequest(paymentDefinitionID));
  } catch (err) {
    next(err);
  }
});

router.post('/', async (req, res, next) => {
  try {
    if (!req.body.name || req.body.name === '') {
      throw new RequestError('Missing or invalid payment definition name', 400);
    }
    if (!utils.regexps.ACCOUNT.test(req.body.author)) {
      throw new RequestError('Missing or invalid payment definition author', 400);
    }
    if(!Number.isInteger(req.body.amount)) {
      throw new RequestError('Missing or invalid payment amount', 400)
    } else if (!(req.body.amount > 0)) {
      throw new RequestError('Payment amount must be greater than 0', 400)
    }
    if (req.body.descriptionSchema && !ajv.validateSchema(req.body.descriptionSchema)) {
      throw new RequestError('Invalid description schema', 400);
    }
    await paymentDefinitionsHandler.handleCreatePaymentDefinitionRequest(req.body.name,
      req.body.author, req.body.amount, req.body.descriptionSchema);
    res.send({ status: 'submitted' });
  } catch (err) {
    next(err);
  }
});

export default router;
