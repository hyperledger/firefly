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

import { ClientEventType } from "../lib/interfaces";
import { config } from '../lib/config';
import { settings } from '../lib/settings';
import * as utils from '../lib/utils';
import * as app2app from '../clients/app2app';

const log = utils.getLogger('handlers/client-events.ts');

export const clientEventHandler = (eventType: ClientEventType, content: object) => {
  if (settings.clientEvents.includes(eventType)) {
    log.info(`Dispatched client event ${eventType}`);
    app2app.dispatchMessage(config.app2app.destinations.client, { type: eventType, content });
  }
};
