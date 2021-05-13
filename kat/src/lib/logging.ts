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

export function setLogLevel(level?: string) {
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
    for (const arg of args) {
      // Special handling of axios errors to avoid massive dumps in log
      if (arg?.isAxiosError) {
        let data = arg.response?.data;
        data = data?.on ? '[stream]' : JSON.stringify(data);
        logArgs.push(`HTTP [${arg.response?.status}] ${arg.message}: ${data}`)
      } else {
        logArgs.push(arg);
      }
    }
    console.log(`${new Date().toISOString()} [${level}]:`, ...logArgs, this.loggerName);
  }
  
}

setLogLevel();
