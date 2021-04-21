import { Router } from 'express';
import * as batchHandler from '../handlers/batches';

const router = Router();

router.get('/:batchID', async (req, res, next) => {
  try {
    res.send(await batchHandler.handleGetBatchRequest(req.params.batchID));
  } catch (err) {
    next(err);
  }
});

export default router;
