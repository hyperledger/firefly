import { Router } from 'express';
import * as membersHandler from '../handlers/members';
import RequestError from '../lib/request-error';
import { constants } from '../lib/utils';

const router = Router();

router.get('/', async (req, res, next) => {
  const skip = Number(req.query.skip || 0);
  const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
  const owned = req.query.owned === 'true';
  try {
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await membersHandler.handleGetMembersRequest(skip, limit, owned));
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
    if (!(req.body.address && req.body.name && req.body.app2appDestination
      && req.body.docExchangeDestination)) {
      throw new RequestError('Invalid member', 400);
    }
    const sync = req.query.sync === 'true';
    await membersHandler.handleUpsertMemberRequest(req.body.address, req.body.name,
      req.body.app2appDestination, req.body.docExchangeDestination, sync);
    res.send({ status: 'submitted' });
  } catch (err) {
    next(err);
  }
});

export default router;