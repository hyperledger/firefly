import { HardhatUserConfig } from 'hardhat/config';
import '@nomiclabs/hardhat-etherscan';
import '@nomiclabs/hardhat-waffle';
import '@typechain/hardhat';
import 'hardhat-gas-reporter';
import 'solidity-coverage';

// You need to export an object to set up your config
// Go to https://hardhat.org/config/ to learn more

const config: HardhatUserConfig = {
  solidity: "0.8.4",
  defaultNetwork: "firefly-cli",
  networks: {
    'firefly-cli': {
      url: "http://127.0.0.1:5100"
    },
    rinkeby: {
        url: `https://rinkeby.infura.io/v3/${process.env.INFURA_PROJECT_ID}`,
        accounts: [`${process.env.RINKEBY_PRIVATE_KEY}`]
    }
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS !== undefined,
    currency: 'USD',
  },
  etherscan: {
    apiKey: process.env.ETHERSCAN_API_KEY,
  },
};

export default config;