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
