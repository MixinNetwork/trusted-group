// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {Integer} from "./libs/Integer.sol";
import {Bytes} from "./libs/Bytes.sol";
import {BLS} from "./libs/BLS.sol";
import {Storage} from "./Storage.sol";
import {IRegistry, Registrable} from "./Registrable.sol";
import {Factory} from "./Factory.sol";
import {Asset} from "./Asset.sol";
import {User} from "./User.sol";

contract Registry is IRegistry, Factory {
    using Bytes for bytes;
    using BLS for uint256[2];
    using BLS for bytes;

    event Halted(bool state);
    event Iterated(uint256[4] from, uint256[4] to);
    event MixinTransaction(bytes raw);
    event MixinEvent(Event evt);

    uint128 public immutable PID;

    uint256[4] public GROUP;
    uint64 public INBOUND = 0;
    uint64 public OUTBOUND = 0;
    bool public HALTED = false;

    mapping(uint128 => uint256) public balances;

    struct Event {
        uint64 nonce;
        address user;
        address asset;
        uint256 amount;
        bytes extra;
        uint64 timestamp;
        uint256[2] sig;
    }

    constructor(bytes memory raw, uint128 pid) {
        require(raw.length == 128);
        require(pid > 0);
        GROUP = [
            raw.toUint256(0),
            raw.toUint256(32),
            raw.toUint256(64),
            raw.toUint256(96)
        ];
        PID = pid;
    }

    function iterate(bytes memory raw) public {
        require(HALTED, "invalid state");
        require(raw.length == 256, "invalid input size");
        uint256[4] memory group = [
            raw.toUint256(0),
            raw.toUint256(32),
            raw.toUint256(64),
            raw.toUint256(96)
        ];
        uint256[2] memory sig1 = [raw.toUint256(128), raw.toUint256(160)];
        uint256[2] memory sig2 = [raw.toUint256(192), raw.toUint256(224)];
        uint256[2] memory message = raw.slice(0, 128).hashToPoint();
        require(sig1.verifySingle(GROUP, message), "invalid signature");
        require(sig2.verifySingle(group, message), "invalid signature");
        emit Iterated(GROUP, group);
        GROUP = group;
    }

    function halt(bytes memory raw) public {
        bytes memory input = bytes("HALT").concat(
            Integer.uint64ToFixedBytes(INBOUND)
        );
        uint256[2] memory sig = [raw.toUint256(0), raw.toUint256(32)];
        uint256[2] memory message = input.hashToPoint();
        require(sig.verifySingle(GROUP, message), "invalid signature");
        HALTED = !HALTED;
        emit Halted(HALTED);
    }

    function claim(address asset, uint256 amount) external returns (bool) {
        require(users[msg.sender].length > 0, "invalid user");
        require(assets[asset] > 0, "invalid asset");
        Asset(asset).burn(msg.sender, amount);
        sendMixinTransaction(msg.sender, asset, amount, new bytes(0));
        return true;
    }

    function burn(
        address user,
        uint256 amount,
        bytes memory extra
    ) external returns (bool) {
        require(assets[msg.sender] > 0, "invalid asset");
        if (users[user].length == 0) {
            return true;
        }
        Asset(msg.sender).burn(user, amount);
        sendMixinTransaction(user, msg.sender, amount, extra);
        return true;
    }

    function sendMixinTransaction(
        address user,
        address asset,
        uint256 amount,
        bytes memory extra
    ) internal {
        uint256 balance = balances[assets[asset]];
        bytes memory log = buildMixinTransaction(
            OUTBOUND,
            users[user],
            assets[asset],
            amount,
            extra
        );
        emit MixinTransaction(log);
        balances[assets[asset]] = balance - amount;
        OUTBOUND = OUTBOUND + 1;
    }

    // process || nonce || asset || amount || extra || timestamp || members || threshold || sig
    function buildMixinTransaction(
        uint64 nonce,
        bytes memory receiver,
        uint128 asset,
        uint256 amount,
        bytes memory extra
    ) internal returns (bytes memory) {
        if (extra.length >= 68 && extra.toUint128(0) == PID) {
            Storage stg = Storage(extra.toAddress(16));
            bytes memory data = extra.slice(68, extra.length - 68);
            stg.write(extra.toUint256(36), data);
            extra = extra.slice(0, 68);
        }

        bytes memory raw = Integer.uint128ToFixedBytes(PID);
        raw = raw.concat(Integer.uint64ToFixedBytes(nonce));
        raw = raw.concat(Integer.uint128ToFixedBytes(asset));
        (bytes memory ab, uint16 al) = Integer.uint256ToVarBytes(amount);
        raw = raw.concat(Integer.uint16ToFixedBytes(al));
        raw = raw.concat(ab);
        raw = raw.concat(Integer.uint16ToFixedBytes(uint16(extra.length)));
        raw = raw.concat(extra);
        raw = raw.concat(Integer.uint64ToFixedBytes(uint64(block.timestamp)));
        raw = raw.concat(receiver);
        raw = raw.concat(new bytes(2));
        return raw;
    }

    // process || nonce || asset || amount || extra || timestamp || members || threshold || sig
    function mixin(bytes memory raw) public returns (bool) {
        require(!HALTED, "invalid state");
        require(raw.length >= 141, "event data too small");

        Event memory evt;
        uint256 offset = 0;

        uint128 id = raw.toUint128(offset);
        require(id == PID, "invalid process");
        offset = offset + 16;

        evt.nonce = raw.toUint64(offset);
        require(evt.nonce == INBOUND, "invalid nonce");
        INBOUND = INBOUND + 1;
        offset = offset + 8;

        (offset, id, evt.amount) = parseEventAsset(raw, offset);
        (offset, evt.extra, evt.timestamp) = parseEventExtra(raw, offset);
        (offset, evt.user) = parseEventUser(raw, offset);
        (evt.asset, evt.extra) = parseEventInput(id, evt.extra);

        offset = offset + 2;
        evt.sig = [raw.toUint256(offset), raw.toUint256(offset + 32)];
        uint256[2] memory message = raw
            .slice(0, offset - 2)
            .concat(new bytes(2))
            .hashToPoint();
        require(evt.sig.verifySingle(GROUP, message), "invalid signature");

        offset = offset + 64;
        require(raw.length == offset, "malformed event encoding");

        uint256 balance = balances[assets[evt.asset]];
        balances[assets[evt.asset]] = balance + evt.amount;

        emit MixinEvent(evt);
        Asset(evt.asset).mint(evt.user, evt.amount);
        return User(evt.user).run(evt.asset, evt.amount, evt.extra);
    }

    function parseEventExtra(bytes memory raw, uint256 offset)
        internal
        pure
        returns (
            uint256,
            bytes memory,
            uint64
        )
    {
        uint256 size = raw.toUint16(offset);
        offset = offset + 2;
        bytes memory extra = raw.slice(offset, size);
        offset = offset + size;
        uint64 timestamp = raw.toUint64(offset);
        offset = offset + 8;
        return (offset, extra, timestamp);
    }

    function parseEventAsset(bytes memory raw, uint256 offset)
        internal
        pure
        returns (
            uint256,
            uint128,
            uint256
        )
    {
        uint128 id = raw.toUint128(offset);
        require(id > 0, "invalid asset");
        offset = offset + 16;
        uint256 size = raw.toUint16(offset);
        offset = offset + 2;
        require(size <= 32, "integer out of bounds");
        uint256 amount = new bytes(32 - size)
            .concat(raw.slice(offset, size))
            .toUint256(0);
        offset = offset + size;
        return (offset, id, amount);
    }

    function parseEventUser(bytes memory raw, uint256 offset)
        internal
        returns (uint256, address)
    {
        uint16 size = raw.toUint16(offset);
        size = 2 + size * 16 + 2;
        bytes memory members = raw.slice(offset, size);
        offset = offset + size;
        address user = getOrCreateUserContract(members);
        return (offset, user);
    }

    function parseEventInput(uint128 id, bytes memory extra)
        internal
        returns (address, bytes memory)
    {
        uint256 offset = 0;
        uint16 size = extra.toUint16(offset);
        offset = offset + 2;
        string memory symbol = string(extra.slice(offset, size));
        offset = offset + size;
        size = extra.toUint16(offset);
        offset = offset + 2;
        string memory name = string(extra.slice(offset, size));
        offset = offset + size;
        bytes memory input = extra.slice(offset, extra.length - offset);
        if (input.length == 68 && input.toUint128(0) == PID) {
            input = Storage(input.toAddress(16)).read(input.toUint256(36));
        }
        address asset = getOrCreateAssetContract(id, symbol, name);
        return (asset, input);
    }
}
