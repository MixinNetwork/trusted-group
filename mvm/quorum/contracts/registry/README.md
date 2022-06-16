verify with the artifacts/build-info json files and use the input part as a single file only

the problem is the BLS library configuration should be on itself, in hardhat.config.js

      libraries: {
        "contracts/BLS.sol": {
          "BLS": "0xFC0105258bf41022AEFbBc8e5671ed97C161CfcC"
        }
      },
