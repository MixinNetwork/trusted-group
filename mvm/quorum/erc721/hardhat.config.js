require('@nomiclabs/hardhat-waffle');
require('hardhat-abi-exporter');
require('solidity-coverage');

const MVM_RPC = "https://geth.mvm.dev";
const MVM_DEPLOYER = process.env.MVM_DEPLOYER; // account private key on MVM

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
  solidity: {
    version: "0.8.9",
    settings: {
      evmVersion: "london",
      metadata: {
        useLiteralContent: true,
        bytecodeHash: "bzzr1"
      },
      optimizer: {
        enabled: true,
        runs: 200
      }
    }
  },
  networks: {
    hardhat: {
      initialBaseFeePerGas: 0 // hardhat london fork error fix for coverage
    },
    mvm: {
      url: `${MVM_RPC}`,
      accounts: [`${MVM_DEPLOYER}`]
    }
  },
  paths: {
    sources: './src/*',
    artifacts: './build',
    tests: './src/tests/*'
  },
  abiExporter: {
    path: './abi',
    clear: true,
    flat: true,
  }
};
