import { start } from './index';
import * as utils from './lib/utils';

const log = utils.getLogger('app.ts');

export const promise = start().catch(err => {
  log.error(`Failed to start asset trail. ${err}`);
});