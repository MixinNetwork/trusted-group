import { task } from "hardhat/config";
import '@nomiclabs/hardhat-ethers';
import '@nomiclabs/hardhat-waffle';
import '@typechain/hardhat';

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
      libraries: {
        //"contracts/libs/BLS.sol": {
          //"BLS": "0x05f24bC12e8F1649FCBf748c1571f549542a9E45"
        //}
      },
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
    mvm: {
      url: `${MVM_RPC}`,
      accounts: [`${MVM_DEPLOYER}`]
    }
  },
  typechain: {
    outDir: 'typechain-types',
    target: 'ethers-v5',
    alwaysGenerateOverloads: false, // should overloads with full signatures like deposit(uint256) be generated always, even if there are no overloads?
    externalArtifacts: ['externalArtifacts/*.json'], // optional array of glob patterns with external artifacts to process (for example external libs from node_modules)
    dontOverrideCompile: false // defaults to true for javascript projects
  },
};
