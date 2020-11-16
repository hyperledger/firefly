import { promisify } from 'util';
import { readFile, writeFile } from 'fs';
import path from 'path';
import * as utils from './utils';
import { createLogger, LogLevelString } from 'bunyan';

const log = createLogger({ name: 'lib/settings.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

const asyncReadFile = promisify(readFile);
const asyncWriteFile = promisify(writeFile);
export let values: {[key: string]: any};

const settingsFilePath = path.join(utils.constants.DATA_DIRECTORY, utils.constants.SETTINGS_FILE_NAME);

export const init = async () => {
  await readSettingsFile()
};

const readSettingsFile = async () => {
  try {
    values = JSON.parse(await asyncReadFile(settingsFilePath, 'utf8'));
  } catch(err) {
    if(err.errno === -2) {
      values = {};
    } else {
      throw new Error(`Failed to read configuration file. ${err}`);
    }
  }
};

export const setSetting = (key: string, value: any) => {
  values[key] = value;
  persistSettings();
};

const persistSettings = () => {
  try {
    asyncWriteFile(settingsFilePath, JSON.stringify(values));
  } catch(err) {
    log.error(`Failed to persist settings. ${err}`);
  }
};