// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {MixinProcess} from './mixin.sol';

// a simple contract to refund everything
contract RefundWorker is MixinProcess {
  // PID is a UUID of Mixin Messenger user, e.g. 27d0c319-a4e3-38b4-93ff-cb45da8adbe1
  uint128 public constant PID = 0x27d0c319a4e338b493ffcb45da8adbe1;

  function _pid() internal pure override(MixinProcess) returns (uint128) {
    return PID;
  }

  // just refund everything
  function _work(Event memory evt) internal override(MixinProcess) returns (bool) {
    require(evt.timestamp > 0, "invalid timestamp");
    require(evt.nonce % 2 == 1, "not an odd nonce");

    bytes memory log = buildMixinTransaction(evt.nonce, evt.asset, evt.amount, evt.extra, evt.members);
    emit MixinTransaction(log);

    return true;
  }
}
