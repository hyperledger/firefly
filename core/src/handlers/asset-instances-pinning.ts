import { createLogger, LogLevelString } from 'bunyan';
import * as apiGateway from '../clients/api-gateway';
import * as ipfs from '../clients/ipfs';
import { BatchManager } from '../lib/batch-manager';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IAssetInstance, IDBBatch, IPinnedBatch } from '../lib/interfaces';
import * as utils from '../lib/utils';

const log = createLogger({ name: 'lib/batch-manager.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

export class AssetInstancesPinning {

  private batchManager = new BatchManager<IAssetInstance>('asset-instances', this.processBatch.bind(this));

  public async init() {
    await this.batchManager.init();
  }

  public async pin(instance: IAssetInstance): Promise<string> {
    const pinnedInstance = { ...instance };
    if (instance.isContentPrivate) delete pinnedInstance.content;
    const batchID = await this.batchManager.getProcessor(instance.author).add(pinnedInstance);
    log.trace(`Pinning initiated for asset ${instance.assetInstanceID}/${instance.assetInstanceID} in batch ${batchID}`);
    return batchID;
  }

  private async processBatch(batch: IDBBatch<IAssetInstance>) {
    // Extract the hashable portion, and write it to IPFS, and store the hash
    const pinnedBatch: IPinnedBatch<IAssetInstance> = {
      type: batch.type,
      created: batch.created,
      author: batch.author,
      completed: batch.completed,
      batchID: batch.batchID,
      records: batch.records,
    };
    batch.batchHash = utils.ipfsHashToSha256(await ipfs.uploadString(JSON.stringify(pinnedBatch)));;

    let apiGatewayResponse: IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse;
    apiGatewayResponse = await apiGateway.createAssetInstanceBatch(batch.batchHash, batch.author, batch.participants);
    batch.receipt = apiGatewayResponse.type === 'async' ? apiGatewayResponse.id : undefined;
  
    // The batch processor who called us does the store back to the local MongoDB, as part of completing the batch
  }

}

/**
 * Singleton instance
 */
export const assetInstancesPinning = new AssetInstancesPinning();
