import { NextFunction, Request, Response } from 'express';
import { createLogger, LogLevelString } from 'bunyan';
import * as utils from './utils';
const log = createLogger({ name: 'lib/request-error.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

export default class RequestError extends Error {

  responseCode: number;

  constructor(message: string, responseCode = 500) {
    super(message);
    this.responseCode = responseCode;
  }

}

export const errorHandler = (err: Error, req: Request, res: Response, _next: NextFunction) => {
  const statusCode = err instanceof RequestError?err.responseCode : 500;
  log.error(`!<-- ${req.method} ${req.url} [${statusCode}]`, statusCode === 404 ? undefined : err.stack);
  res.status(statusCode).send({ error: err.message });
};