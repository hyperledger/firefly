// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as apiGateway from '../clients/api-gateway';
import * as ipfs from '../clients/ipfs';
import { BatchManager } from '../lib/batch-manager';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse, IAssetInstance, IAssetInstancePropertySet, IBatchRecord, IDBBatch, IPinnedBatch, BatchRecordType } from '../lib/interfaces';
import * as utils from '../lib/utils';

const log = utils.getLogger('lib/asset-instance-pinning.ts');

export class AssetInstancesPinning {

  private batchManager = new BatchManager('asset-instances', this.processBatch.bind(this));

  public async init() {
    await this.batchManager.init();
  }

  public async pin(instance: IAssetInstance): Promise<string> {
    const pinnedInstance: IBatchRecord = { recordType: BatchRecordType.assetInstance, ...instance };
    if (instance.isContentPrivate) delete pinnedInstance.content;
    const batchID = await this.batchManager.getProcessor(instance.author).add(pinnedInstance);
    log.trace(`Pinning initiated for asset ${instance.assetInstanceID}/${instance.assetInstanceID} in batch ${batchID}`);
    return batchID;
  }

  public async pinProperty(property: IAssetInstancePropertySet): Promise<string> {
    const pinnedProperty: IBatchRecord = { recordType: BatchRecordType.assetProperty, ...property };
    const batchID = await this.batchManager.getProcessor(property.author).add(pinnedProperty);
    log.trace(`Pinning initiated for property ${property.assetInstanceID}/${property.assetInstanceID}/${property.key} in batch ${batchID}`);
    return batchID;
  }

  private async processBatch(batch: IDBBatch) {
    // Extract the hashable portion, and write it to IPFS, and store the hash
    const pinnedBatch: IPinnedBatch = {
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
