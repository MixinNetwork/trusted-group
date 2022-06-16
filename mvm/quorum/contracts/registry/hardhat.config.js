require("@nomiclabs/hardhat-waffle");

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
  }
};
