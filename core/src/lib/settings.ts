import Ajv from 'ajv';
import { promisify } from 'util';
import { readFile, writeFile } from 'fs';
import path from 'path';
import * as utils from './utils';
import { createLogger, LogLevelString } from 'bunyan';
import { ISettings } from './interfaces';
import settingsSchema from '../schemas/settings.json';

const log = createLogger({ name: 'lib/settings.ts', level: utils.constants.LOG_LEVEL as LogLevelString });

const asyncReadFile = promisify(readFile);
const asyncWriteFile = promisify(writeFile);
const ajv = new Ajv();
const validateSettings = ajv.compile(settingsSchema);
export let settings: ISettings;

const settingsFilePath = path.join(utils.constants.DATA_DIRECTORY, utils.constants.SETTINGS_FILE_NAME);

export const init = async () => {
  await readSettingsFile()
};

const readSettingsFile = async () => {
  try {
    const values = JSON.parse(await asyncReadFile(settingsFilePath, 'utf8'));
    if(!validateSettings(values)) {
      throw new Error('Invalid content');
    }
    settings = values;
  } catch(err) {
    if(err.errno === -2) {
      settings = {
        clientEvents: []
      };
    } else {
      throw new Error(`Failed to read settings file. ${err}`);
    }
  }
};

export const setClientEvents = (clientEvents: string[]) => {
  settings.clientEvents = clientEvents;
  persistSettings();
};

const persistSettings = () => {
  try {
    asyncWriteFile(settingsFilePath, JSON.stringify(settings));
  } catch(err) {
    log.error(`Failed to persist settings. ${err}`);
  }
};