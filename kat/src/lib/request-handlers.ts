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