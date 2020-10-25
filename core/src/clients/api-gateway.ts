import axios from 'axios';
import { config } from '../lib/config';

export const upsertMember = async (address: string, name: string, app2appDestination: string,
  docExchangeDestination: string, sync = false) => {
  await axios({
    method: 'post',
    url: `${config.apiGateway.apiEndpoint}/registerMember?kld-from=${address}&kld-sync=${sync}`,
    auth: {
      username: config.appCredentials.user,
      password: config.appCredentials.password
    },
    data: { name, app2appDestination, docExchangeDestination }
  });
};
