import { ClientEventType } from "../lib/interfaces";
import { config } from '../lib/config';
import { settings } from '../lib/settings';
import * as utils from '../lib/utils';
import * as app2app from '../clients/app2app';
import { createLogger, LogLevelString } from 'bunyan';

const log = createLogger({ name: 'handlers/client-events.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

export const clientEventHandler = (eventType: ClientEventType, content: object) => {
  if (settings.clientEvents.includes(eventType)) {
    log.trace(`Dispatched client event ${eventType}`);
    app2app.dispatchMessage(config.app2app.destinations.client, { type: eventType, content });
  }
};
