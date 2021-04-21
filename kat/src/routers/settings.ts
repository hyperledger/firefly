import { Router } from 'express';
import * as settings from '../lib/settings';
import RequestError from '../lib/request-handlers';

const router = Router();

router.get('/', async (_req, res, next) => {
  try {
    res.send(settings.settings);
  } catch (err) {
    next(err);
  }
});

router.put('/', async (req, res, next) => {
  try {
    if (req.body.key === undefined || req.body.value === undefined) {
      throw new RequestError('Invalid setting key/value', 400);
    }
    await settings.updateSettings(req.body.key, req.body.value);
    res.send({ status: 'success' });
  } catch (err) {
    next(err);
  }
});

export default router;