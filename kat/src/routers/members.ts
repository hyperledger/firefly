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

import { Router } from 'express';
import * as membersHandler from '../handlers/members';
import { config } from '../lib/config';
import RequestError from '../lib/request-handlers';
import { constants } from '../lib/utils';

const router = Router();

router.get('/', async (req, res, next) => {
  try {
    const skip = Number(req.query.skip || 0);
    const limit = Number(req.query.limit || constants.DEFAULT_PAGINATION_LIMIT);
    if (isNaN(skip) || isNaN(limit)) {
      throw new RequestError('Invalid skip / limit', 400);
    }
    res.send(await membersHandler.handleGetMembersRequest({}, skip, limit));
  } catch (err) {
    next(err);
  }
});

router.get('/:memberAddress', async (req, res, next) => {
  try {
    res.send(await membersHandler.handleGetMemberRequest(req.params.memberAddress));
  } catch (err) {
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
    let assetTrailInstanceID, app2appDestination, docExchangeDestination;
    switch (config.protocol) {
      case 'corda':
        if (!req.body.assetTrailInstanceID) {
          throw new RequestError('Missing member assetTrailInstanceID', 400);
        }
        if (!req.body.app2appDestination) {
          throw new RequestError('Missing member app2appDestination', 400);
        }
        if (!req.body.docExchangeDestination) {
          throw new RequestError('Missing member docExchangeDestination', 400);
        }
        assetTrailInstanceID = req.body.assetTrailInstanceID;
        app2appDestination = req.body.app2appDestination;
        docExchangeDestination = req.body.docExchangeDestination;
        break;
      case 'ethereum':
        assetTrailInstanceID = config.assetTrailInstanceID;
        app2appDestination = config.app2app.destinations.kat;
        docExchangeDestination = config.docExchange.destination;
        break;
    }
    const sync = req.query.sync === 'true';
    await membersHandler.handleUpsertMemberRequest(req.body.address, req.body.name, assetTrailInstanceID, app2appDestination, docExchangeDestination, sync);
    res.send({ status: sync ? 'success' : 'submitted' });
  } catch (err) {
    next(err);
  }
});

export default router;