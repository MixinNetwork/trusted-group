// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {BytesLib} from './bytes.sol';
import {BLS} from './bls.sol';

abstract contract MixinProcess {
  using BytesLib for bytes;
  using BLS for uint256[2];
  using BLS for bytes;

  struct Event {
    uint128 process;
    uint64 nonce;
    uint128 asset;
    uint256 amount;
    bytes extra;
    uint64 timestamp;
    bytes members;
    address sender;
    uint256[2] sig;
  }

  event MixinTransaction(bytes);
  event MixinEvent(address indexed sender, uint256 nonce, uint128 asset, uint256 amount, uint64 timestamp, bytes extra);

  uint256[4] public GROUP = [
    0x2f741961cea2e88cfa2680eeaac040d41f41f3fedb01e38c06f4c6058fd7e425, // x.y
    0x007d68aef83f9690b04f463e13eadd9b18f4869041f1b67e7f1a30c9d1d2c42c, // x.x
    0x2a32fa1736807486256ad8dc6a8740dfb91917cf8d15848133819275be92b673, // y.y
    0x257ad901f02f8a442ccf4f1b1d0d7d3a8e8fe791102706e575d36de1c2a4a40f  // y.x
  ];

  uint64 public NONCE = 0;
  mapping(uint128 => uint256) public custodian;
  mapping(address => bytes) public members;

  // the contract should return a valid PID
  function _pid() internal pure virtual returns (uint128);

  // the contract should implement this method
  function _work(Event memory evt) internal virtual returns (bool);

  function work(Event memory evt) external returns (bool) {
    require(msg.sender == address(this), "invalid work commander");
    return _work(evt);
  }

  // process || nonce || asset || amount || extra || timestamp || members || threshold || sig
  function mixin(bytes calldata raw) public returns (bool) {
    require(_pid() > 0, "invalid pid");
    require(raw.length >= 141, "event data too small");

    Event memory evt;

    uint256 size = 0;
    uint256 offset = 0;
    evt.process = raw.toUint128(offset);
    require(evt.process == _pid(), "invalid process");
    offset = offset + 16;

    evt.nonce = raw.toUint64(offset);
    require(evt.nonce == NONCE, "invalid nonce");
    NONCE = NONCE + 1;
    offset = offset + 8;

    evt.asset = raw.toUint128(offset);
    offset = offset + 16;

    size = raw.toUint16(offset);
    offset = offset + 2;
    require(size <= 32, "integer out of bounds");
    evt.amount = new bytes(32 - size).concat(raw.slice(offset, size)).toUint256(0);
    offset = offset + size;

    size = raw.toUint16(offset);
    offset = offset + 2;
    evt.extra = raw.slice(offset, size);
    offset = offset + size;

    evt.timestamp = raw.toUint64(offset);
    offset = offset + 8;

    size = raw.toUint16(offset);
    size = 2 + size * 16 + 2;
    evt.members = raw.slice(offset, size);
    evt.sender = mixinSenderToAddress(evt.members);
    offset = offset + size;

    offset = offset + 2;
    evt.sig = [raw.toUint256(offset), raw.toUint256(offset+32)];
    uint256[2] memory message = raw.slice(0, offset-2).concat(new bytes(2)).hashToPoint();
    require(evt.sig.verifySingle(GROUP, message), "invalid signature");

    offset = offset + 64;
    require(raw.length == offset, "malformed event encoding");

    custodian[evt.asset] = custodian[evt.asset] + evt.amount;
    members[evt.sender] = evt.members;

    emit MixinEvent(evt.sender, evt.nonce, evt.asset, evt.amount, evt.timestamp, evt.extra);
    try this.work(evt) returns (bool result) {
      return result;
    } catch {
      bytes memory log = buildMixinTransaction(evt.nonce, evt.asset, evt.amount, evt.extra, evt.members);
      emit MixinTransaction(log);
      return false;
    }
  }

  // pid || nonce || asset || amount || extra || timestamp || members || threshold || sig
  function buildMixinTransaction(uint64 nonce, uint128 asset, uint256 amount, bytes memory extra, bytes memory receiver) internal returns (bytes memory) {
    require(extra.length < 128, "extra too large");
    require(custodian[asset] >= amount, "insufficient custodian");
    custodian[asset] = custodian[asset] - amount;
    bytes memory raw = uint128ToFixedBytes(_pid());
    raw = raw.concat(uint64ToFixedBytes(nonce));
    raw = raw.concat(uint128ToFixedBytes(asset));
    (bytes memory ab, uint16 al) = uint256ToVarBytes(amount);
    raw = raw.concat(uint16ToFixedBytes(al));
    raw = raw.concat(ab);
    raw = raw.concat(uint16ToFixedBytes(uint16(extra.length)));
    raw = raw.concat(extra);
    raw = raw.concat(new bytes(8));
    raw = raw.concat(receiver);
    raw = raw.concat(new bytes(2));
    return raw;
  }

  function mixinSenderToAddress(bytes memory sender) internal pure returns (address) {
    return address(uint160(uint256(keccak256(sender))));
  }

  function uint16ToFixedBytes(uint16 x) internal pure returns (bytes memory) {
    bytes memory c = new bytes(2);
    bytes2 b = bytes2(x);
    for (uint i=0; i < 2; i++) {
      c[i] = b[i];
    }
    return c;
  }

  function uint64ToFixedBytes(uint64 x) internal pure returns (bytes memory) {
    bytes memory c = new bytes(8);
    bytes8 b = bytes8(x);
    for (uint i=0; i < 8; i++) {
      c[i] = b[i];
    }
    return c;
  }

  function uint128ToFixedBytes(uint128 x) internal pure returns (bytes memory) {
    bytes memory c = new bytes(16);
    bytes16 b = bytes16(x);
    for (uint i=0; i < 16; i++) {
      c[i] = b[i];
    }
    return c;
  }

  function uint256ToVarBytes(uint256 x) internal pure returns (bytes memory, uint16) {
    bytes memory c = new bytes(32);
    bytes32 b = bytes32(x);
    uint16 offset = 0;
    for (uint16 i=0; i < 32; i++) {
      c[i] = b[i];
      if (c[i] > 0 && offset == 0) {
        offset = i;
      }
    }
    uint16 size = 32 - offset;
    return (c.slice(offset, 32-offset), size);
  }
}
