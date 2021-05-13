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

import * as database from '../clients/database';
import { BatchProcessor } from './batch-processor';
import { IDBBatch } from './interfaces';
import * as utils from './utils';

const log = utils.getLogger('lib/batch-manager.ts');

/**
 * Lifecycle manager for BatchProcessor instances, within a single type, across multiple authors
 */
export class BatchManager {

  processors: {[author: string]: BatchProcessor} = {};

  constructor(
    private type: string,
    private processBatchCallback: (batch: IDBBatch) => Promise<void>,
  ) { }

  public async init() {
    // Query all incomplete batches for our type, in creation order
    const inflightBatches = await database.retrieveBatches({
      type: this.type,
      completed: null,
    }, 0, 0, { created: 1 });
    const byAuthor: {[author: string]: IDBBatch[]} = {};
    for (const inflight of inflightBatches) {
      const forAuthor = byAuthor[inflight.author] = byAuthor[inflight.author] || [];
      forAuthor.push(inflight);
    }
    // Init a set of processors for each distinct authors.
    // Note these will be reaped if no work comes in while we're processing the backlog
    for (const [author, forAuthor] of Object.entries(byAuthor)) {
      await this.getProcessor(author).init(forAuthor);
    }
  }

  protected processorCompleteCallback(author: string) {
    log.trace(`${this.type} batch manager: Reaping processor for ${author}`);
    delete this.processors[author];
  }

  public getProcessor(author: string) {
    if (!this.processors[author]) {
      log.trace(`${this.type} batch manager: Creating processor for ${author}`);
      this.processors[author] = new BatchProcessor(
        author,
        this.type,
        this.processBatchCallback,
        this.processorCompleteCallback.bind(this),
      );
    }
    return this.processors[author];
  }

}