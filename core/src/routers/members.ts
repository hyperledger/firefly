import { Router } from 'express';
import * as membersHandler from '../handlers/members';
import RequestError from '../lib/request-error';
import { constants } from '../lib/utils';

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
    res.send(await membersHandler.handleGetMembersRequest(query, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:memberAddress', async (req, res, next) => {
  try {
    res.send(await membersHandler.handleGetMemberRequest(req.params.memberAddress));
  } catch(err) {
    next(err)
  }
});

router.put('/', async (req, res, next) => {
  try {
    if (!req.body.address) {
      throw new RequestError('Missing member address', 400);
    }
    if (!req.body.name) {
      throw new RequestError('Missing member name', 400);
    }
    const sync = req.query.sync === 'true';
    await membersHandler.handleUpsertMemberRequest(req.body.address, req.body.name, sync);
    res.send({ status: sync? 'success' : 'submitted' });
  } catch (err) {
    next(err);
  }
});

export default router;