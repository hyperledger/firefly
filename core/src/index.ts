import { initConfig, config } from './lib/config';
import express from 'express';
import bodyParser from 'body-parser';
import apiRouter from './routers/api';
import { errorHandler } from './lib/request-error';
import * as ipfs from './clients/ipfs';
import * as docExchange from './clients/doc-exchange';
import * as eventStreams from './clients/event-streams';

initConfig()
  .then(() => ipfs.init())
  .then(() => docExchange.init())
  .then(() => {
    eventStreams.init();
    const app = express();

    app.use(bodyParser.urlencoded({ extended: true }));
    app.use(bodyParser.json());

    app.use('/api/v1', apiRouter);
    app.use(errorHandler);

    app.listen(config.port, () => {
      console.log(`Asset trail listening on port ${config.port}`);
    });

  })
  .catch(err => {
    console.log(`Failed to start asset trail. ${err}`);
  });