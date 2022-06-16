// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import { BLS } from "../libs/BLS.sol";

contract TestBLS {
  function verifySingle(
    uint256[2] calldata signature,
    uint256[4] calldata pubkey,
    uint256[2] calldata message
  ) external view returns (bool) {
    return BLS.verifySingle(signature, pubkey, message);
  }
}
