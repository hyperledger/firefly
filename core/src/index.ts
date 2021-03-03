import { config, init as initConfig, shutDown as shutDownConfig } from './lib/config';
import express from 'express';
import bodyParser from 'body-parser';
import membersRouter from './routers/members';
import assetDefinitionsRouter from './routers/asset-definitions';
import assetInstancesRouter from './routers/asset-instances';
import paymentDefinitionsRouter from './routers/payment-definitions';
import paymentInstancesRouter from './routers/payment-instances';
import settingsRouter from './routers/settings';
import batchesRouter from './routers/batches';
import { errorHandler } from './lib/request-error';
import * as database from './clients/database';
import * as settings from './lib/settings';
import * as utils from './lib/utils';
import * as ipfs from './clients/ipfs';
import * as app2app from './clients/app2app';
import * as docExchange from './clients/doc-exchange';
import * as eventStreams from './clients/event-streams';
import { assetTradeHandler } from './handlers/asset-trade';
import { clientEventHandler } from './handlers/client-events';
import { createLogger, LogLevelString } from 'bunyan';
import { assetInstancesPinning } from './handlers/asset-instances-pinning';

const log = createLogger({ name: 'index.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

export const start = () => {
  return initConfig(() => { app2app.reset(); docExchange.reset() })
    .then(() => settings.init())
    .then(() => database.init())
    .then(() => ipfs.init())
    .then(() => app2app.init())
    .then(() => docExchange.init())
    .then(() => assetInstancesPinning.init())
    .then(() => {
      eventStreams.init();
      const app = express();

      app.use(bodyParser.urlencoded({ extended: true }));
      app.use(bodyParser.json());

      app.use('/api/v1/members', membersRouter);
      app.use('/api/v1/assets/definitions', assetDefinitionsRouter);
      app.use('/api/v1/assets', assetInstancesRouter);
      app.use('/api/v1/payments/definitions', paymentDefinitionsRouter);
      app.use('/api/v1/payments/instances', paymentInstancesRouter);
      app.use('/api/v1/settings', settingsRouter);
      app.use('/api/v1/batches', batchesRouter);

      app.use(errorHandler);

      app2app.addListener(assetTradeHandler);
      database.addListener(clientEventHandler);

      const server = app.listen(config.port, () => {
        log.info(`Asset trail listening on port ${config.port} - log level "${utils.constants.LOG_LEVEL}"`);
      }).on('error', (err) => {
        log.error(err);
      });

      const shutDown = () => {
        server.close(err => {
          if (err) {
            log.error(`Error closing server. ${err}`);
          } else {
            log.info(`Stopped server.`)
          }
        });
        eventStreams.shutDown();
        database.shutDown();
        shutDownConfig();
      };

      return { app, shutDown };

    });
}
