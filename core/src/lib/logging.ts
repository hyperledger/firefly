"use strict";

const LEVEL_NONE  = 0;
const LEVEL_ERROR = 1;
const LEVEL_WARN  = 2;
const LEVEL_INFO  = 3;
const LEVEL_DEBUG = 4;

const LEVEL_TAGS = {
  [LEVEL_NONE]:  'NONE',
  [LEVEL_ERROR]: 'ERROR',
  [LEVEL_WARN]:  'WARN ',
  [LEVEL_INFO]:  'INFO ',
  [LEVEL_DEBUG]: 'DEBUG',
};

let logLevel = LEVEL_ERROR;

function setLogLevel(level?: string) {
  if (!level) level = process.env.LOG_LEVEL || 'info';
  for (let [l,t] of Object.entries(LEVEL_TAGS)) {
    if (t.trim().toLowerCase() === level.trim().toLowerCase()) {
      logLevel = Number(l);
    }
  }
}

export class Logger {

  constructor(private loggerName?: string) {}

  error() { logLevel >= LEVEL_ERROR && this.log('ERROR', ...arguments); }
  warn()  { logLevel >= LEVEL_WARN  && this.log('WARN ', ...arguments); }
  info()  { logLevel >= LEVEL_INFO  && this.log('INFO ', ...arguments); }
  debug() { logLevel >= LEVEL_DEBUG && this.log('DEBUG', ...arguments); }
  
  private log(level: string, ...args: any[]) {
    console.log(`${new Date().toISOString()} [${level}]:`, ...args, this.loggerName);
  }
  
}

setLogLevel();
