import { Router } from 'express';
import * as membersHandler from '../handlers/members';
import RequestError from '../lib/request-error';
import { constants } from '../lib/utils';

const router = Router();

router.get('/members', async (req, res, next) => {
  const skip = Number(req.query.skip || 0);
  const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
  try {
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await membersHandler.handleGetMembersRequest(skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/members/:member', async (req, res, next) => {
  try {
    const member = await membersHandler.handleGetMemberRequest(req.params.member);
    if(member === null) {
      throw new RequestError('Member not found', 404);
    }
    res.send(member);
  } catch(err) {
    next(err)
  }
});

router.post('/members', async (req, res, next) => {
  try {
    if (!(req.body.address && req.body.name && req.body.app2appDestination
      && req.body.docExchangeDestination)) {
      throw new RequestError('Invalid member', 400);
    }
    await membersHandler.handleUpsertMemberRequest(req.body.address, req.body.name,
      req.body.app2appDestination, req.body.docExchangeDestination);
    res.send({ status: 'submitted' });
  } catch (err) {
    next(err);
  }
});

router.get('/assets/definitions', (req, res) => {
  res.send('ok')
});

router.post('/assets/instances', (req, res) => {
  res.send('ok')
});

export default router;