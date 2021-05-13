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

import assert from 'assert';
import sinon, { SinonStub } from 'sinon';
import * as database from '../../../clients/database';
import { BatchManager } from '../../../lib/batch-manager';
import { BatchProcessor } from '../../../lib/batch-processor';

export const testBatchManager = async () => {

describe('BatchManager', () => {

  let batchRetriever: SinonStub;
  let processorInit: SinonStub;

  beforeEach(() => {
    batchRetriever = sinon.stub(database, 'retrieveBatches');
    processorInit = sinon.stub(BatchProcessor.prototype, 'init');
  });

  afterEach(() => {
    batchRetriever.restore();
    processorInit.restore();
  })

  it('inits all the right processors', async () => {

    batchRetriever.resolves([
      {
        type: 't1',
        author: 'author1',
        batchID: 'author1-batch1',
      },
      {
        type: 't1',
        author: 'author2',
        batchID: 'author2-batch1',
      },
      {
        type: 't1',
        author: 'author3',
        batchID: 'author3-batch1',
      },
      {
        type: 't1',
        author: 'author2',
        batchID: 'author2-batch2',
      },
      {
        type: 't1',
        author: 'author1',
        batchID: 'author1-batch2',
      },
    ]);

    const bm = new BatchManager('t1', sinon.stub());
    await bm.init();

    sinon.assert.calledWith(processorInit, [
      {
        type: 't1',
        author: 'author1',
        batchID: 'author1-batch1',
      },
      {
        type: 't1',
        author: 'author1',
        batchID: 'author1-batch2',
      }
    ]);

    sinon.assert.calledWith(processorInit, [
      {
        type: 't1',
        author: 'author2',
        batchID: 'author2-batch1',
      },
      {
        type: 't1',
        author: 'author2',
        batchID: 'author2-batch2',
      }
    ]);

    sinon.assert.calledWith(processorInit, [
      {
        type: 't1',
        author: 'author3',
        batchID: 'author3-batch1',
      }
    ]);
  });

  it('caches batch processors', async () => {

    class TestBatchManagerWrapper extends BatchManager {
      public processorCompleteCallback(author: string) {
        return super.processorCompleteCallback(author);
      }
    }

    const bm = new TestBatchManagerWrapper('t1', sinon.stub());
    const bp1 = bm.getProcessor('author1');

    // Check it caches
    assert(bp1 === bm.getProcessor('author1'));

    // Check it doesn't clear the wrong entry
    bm.processorCompleteCallback('author2');
    assert(bp1 === bm.getProcessor('author1'));

    // Check it clears the right entry
    bm.processorCompleteCallback('author1');
    assert(bp1 !== bm.getProcessor('author1'));

  })

});
};
