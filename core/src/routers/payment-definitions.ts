import { Router } from 'express';
import RequestError from '../lib/request-error';
import * as paymentDefinitionsHandler from '../handlers/payment-definitions';
import { constants } from '../lib/utils';
import * as utils from '../lib/utils';
import Ajv from 'ajv';

const router = Router();

const ajv = new Ajv();
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
    res.send(await paymentDefinitionsHandler.handleGetPaymentDefinitionsRequest(query, skip, limit));
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

router.post('/', async (req, res, next) => {
  try {
    if (!req.body.name || req.body.name === '') {
      throw new RequestError('Missing or invalid payment definition name', 400);
    }
    if (!utils.regexps.ACCOUNT.test(req.body.author)) {
      throw new RequestError('Missing or invalid payment definition author', 400);
    }
    if (req.body.descriptionSchema && !ajv.validateSchema(req.body.descriptionSchema)) {
      throw new RequestError('Invalid description schema', 400);
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
