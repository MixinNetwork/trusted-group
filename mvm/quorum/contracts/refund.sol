// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {MixinProcess} from './mixin.sol';

// a simple contract to refund everything
contract RefundWorker is MixinProcess {
  // PID is the app id in Mixin which the contract will process, e.g. 27d0c319-a4e3-38b4-93ff-cb45da8adbe1
  // The app id will add 0x as prefix and delete '-'
  uint128 public constant PID = 0x27d0c319a4e338b493ffcb45da8adbe1;

  function _pid() internal pure override(MixinProcess) returns (uint128) {
    return PID;
  }

  // just refund everything
  function _work(Event memory evt) internal override(MixinProcess) returns (bool) {
    require(evt.timestamp > 0, "invalid timestamp");

    address sender = mixinSenderToAddress(evt.sender);
    bytes memory log = encodeMixinEvent(evt.nonce, evt.asset, evt.amount, evt.extra, members[sender]);
    emit MixinTransaction(log);

    return true;
  }
}
