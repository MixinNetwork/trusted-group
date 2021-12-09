// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {MixinProcess} from './mixin.sol';

// a simple contract to refund everything
contract RefundWorker is MixinProcess {
  // PID is the app id in Mixin which the contract will process, e.g. c6d0c728-2624-429b-8e0d-d9d19b6592fa
  // The app id will add 0x as prefix and delete '-'
  uint128 public constant PID = 0x4cfc560f993a3fef92d14f92f7a0b662;

  function _pid() internal pure override(MixinProcess) returns (uint128) {
    return PID;
  }

  // just refund everything
  function _work(address sender, uint64 nonce, uint128 asset, uint256 amount, uint64 timestamp, bytes memory extra) internal override(MixinProcess) returns (bool) {
    require(timestamp > 0, "invalid timestamp");

    bytes memory log = encodeMixinEvent(nonce, asset, amount, extra, members[sender]);
    emit MixinTransaction(log);

    return true;
  }
}
