"use strict";

const LEVEL_NONE  = 0;
const LEVEL_ERROR = 1;
const LEVEL_WARN  = 2;
const LEVEL_INFO  = 3;
const LEVEL_DEBUG = 4;
const LEVEL_TRACE = 5;

const LEVEL_TAGS = {
  [LEVEL_NONE]:  'NONE',
  [LEVEL_ERROR]: 'ERROR',
  [LEVEL_WARN]:  'WARN ',
  [LEVEL_INFO]:  'INFO ',
  [LEVEL_DEBUG]: 'DEBUG',
  [LEVEL_TRACE]: 'TRACE',
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

  error(...args: any[]) { logLevel >= LEVEL_ERROR && this.log('ERROR', ...args); }
  warn(...args: any[])  { logLevel >= LEVEL_WARN  && this.log('WARN ', ...args); }
  info(...args: any[])  { logLevel >= LEVEL_INFO  && this.log('INFO ', ...args); }
  debug(...args: any[]) { logLevel >= LEVEL_DEBUG && this.log('DEBUG', ...args); }
  trace(...args: any[]) { logLevel >= LEVEL_TRACE && this.log('TRACE', ...args); }
  
  private log(level: string, ...args: any[]) {
    const logArgs = [];
    for (const arg of args || []) {
      // Special handling of axios errors to avoid massive dumps in log
      if (arg?.isAxiosError) {
        let data = arg?.response?.data;
        data = data.on ? '[stream]' : JSON.stringify(data);
        logArgs.push(`HTTP [${arg?.response?.status}] ${arg.message}: ${data}`)
      } else {
        logArgs.push(arg);
      }
    }
    console.log(`${new Date().toISOString()} [${level}]:`, ...logArgs, this.loggerName);
  }
  
}

setLogLevel();
