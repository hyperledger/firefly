import { NextFunction, Request, Response } from 'express';
import { nanoid } from 'nanoid';
import * as utils from './utils';
const log = utils.getLogger('lib/request-handlers.ts');

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

interface ATRequest extends Request {
  entryTime: number
  requestId: string
}

export function requestLogger(req: Request, _: Response, next: NextFunction): void {
  const r: ATRequest = req as ATRequest;
  r.entryTime = Date.now();
  r.requestId = nanoid(10);
  log.info(`[${r.requestId}] --> ${req.method} ${req.path}`);
  next();
}

export function responseLogger(req: Request, res: Response, next: NextFunction): void {
  const r: ATRequest = req as ATRequest;
  const timing = (Date.now() - r.entryTime).toFixed(1);
  log.info(
    `[${r.requestId}] <-- ${req.method} ${req.path} [${res.statusCode}] (${timing}ms)`
  );
  next();
}