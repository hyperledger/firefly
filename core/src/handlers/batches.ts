import * as database from '../clients/database';
import RequestError from '../lib/request-handlers';

export const handleGetBatchRequest = async (batchID: string) => {
  const batch = await database.retrieveBatchByID(batchID);
  if (batch === null) {
    throw new RequestError('Asset instance not found', 404);
  }
  return batch;
};
