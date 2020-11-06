import { NextFunction, Request, Response } from 'express';

export default class RequestError extends Error {

  responseCode: number;

  constructor(message: string, responseCode = 500) {
    super(message);
    this.responseCode = responseCode;
  }

}

export const errorHandler = (err: Error, _req: Request, res: Response, _next: NextFunction) => {
  res.status(err instanceof RequestError? err.responseCode : 500).send({ error: err.message });
};