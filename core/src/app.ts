import { createLogger, LogLevelString } from 'bunyan';
import { start } from './index';
import * as utils from './lib/utils';

const log = createLogger({ name: 'index.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

export const promise = start().catch(err => {
  log.error(`Failed to start asset trail. ${err}`);
});