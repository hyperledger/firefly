import { initConfig, config } from './lib/config';

initConfig()
  .then(() => {
    console.log('Asset trail core started');
  })
  .catch(err => {
    console.log(`Failed to start asset train core. ${err}`);
  });