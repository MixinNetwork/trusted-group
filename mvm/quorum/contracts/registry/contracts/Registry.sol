// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {Bytes} from './libs/Bytes.sol';
import {BLS} from './libs/BLS.sol';
import {Storage} from './Storage.sol';
import {IRegistry,Registrable} from './Registrable.sol';
import {Asset} from './Asset.sol';
import {User} from './User.sol';

contract Registry is IRegistry {
    using Bytes for bytes;
    using BLS for uint256[2];
    using BLS for bytes;

    event UserCreated(address at, bytes members);
    event AssetCreated(address at, uint id);
    event Halted(bool state);
    event Iterated(uint256[4] from, uint256[4] to);
    event MixinTransaction(bytes);
    event MixinEvent(Event evt);

    uint256 public constant VERSION = 1;
    uint128 public immutable PID;
    uint256 constant BALANCE = 1;

    uint256[4] public GROUP;
    uint64 public INBOUND = 0;
    uint64 public OUTBOUND = 0;
    bool public HALTED = false;

    mapping(address => bytes) public users;
    mapping(address => uint128) public assets;
    mapping(uint => address) public contracts;
    mapping(uint128 => uint256) public balances;
    address[] public addresses;
    uint128[] public deposits;

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
        uint256[4] memory group = [raw.toUint256(0), raw.toUint256(32), raw.toUint256(64), raw.toUint256(96)];
        uint256[2] memory sig1 = [raw.toUint256(128), raw.toUint256(160)];
        uint256[2] memory sig2 = [raw.toUint256(192), raw.toUint256(224)];
        uint256[2] memory message = raw.slice(0, 128).hashToPoint();
        require(sig1.verifySingle(GROUP, message), "invalid signature");
        require(sig2.verifySingle(group, message), "invalid signature");
        emit Iterated(GROUP, group);
        GROUP = group;
    }

    function halt(bytes memory raw) public {
        bytes memory input = bytes("HALT").concat(uint64ToFixedBytes(INBOUND));
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
        sendMixinTransaction(msg.sender, asset, amount);
        return true;
    }

    function burn(address user, uint256 amount) external returns (bool) {
        require(assets[msg.sender] > 0, "invalid asset");
        if (users[user].length == 0) {
            return true;
        }
        Asset(msg.sender).burn(user, amount);
        sendMixinTransaction(user, msg.sender, amount);
        return true;
    }

    function sendMixinTransaction(address user, address asset, uint256 amount) internal {
        uint256 balance = balances[assets[asset]];
        bytes memory extra = new bytes(0);
        bytes memory log = buildMixinTransaction(OUTBOUND, users[user], assets[asset], amount, extra);
        emit MixinTransaction(log);
        balances[assets[asset]] = balance - amount;
        OUTBOUND = OUTBOUND + 1;
    }

    // process || nonce || asset || amount || extra || timestamp || members || threshold || sig
    function buildMixinTransaction(uint64 nonce, bytes memory receiver, uint128 asset, uint256 amount, bytes memory extra) internal view returns (bytes memory) {
        require(extra.length < 128, "extra too large");
        bytes memory raw = uint128ToFixedBytes(PID);
        raw = raw.concat(uint64ToFixedBytes(nonce));
        raw = raw.concat(uint128ToFixedBytes(asset));
        (bytes memory ab, uint16 al) = uint256ToVarBytes(amount);
        raw = raw.concat(uint16ToFixedBytes(al));
        raw = raw.concat(ab);
        raw = raw.concat(uint16ToFixedBytes(uint16(extra.length)));
        raw = raw.concat(extra);
        raw = raw.concat(uint64ToFixedBytes(uint64(block.timestamp)));
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
        evt.sig = [raw.toUint256(offset), raw.toUint256(offset+32)];
        uint256[2] memory message = raw.slice(0, offset-2).concat(new bytes(2)).hashToPoint();
        require(evt.sig.verifySingle(GROUP, message), "invalid signature");

        offset = offset + 64;
        require(raw.length == offset, "malformed event encoding");

        uint256 balance = balances[assets[evt.asset]];
        if (balance == 0) {
            deposits.push(assets[evt.asset]);
            balance = BALANCE;
        }
        balances[assets[evt.asset]] = balance + evt.amount;

        emit MixinEvent(evt);
        Asset(evt.asset).mint(evt.user, evt.amount);
        return User(evt.user).run(evt.asset, evt.amount, evt.extra);
    }

    function parseEventExtra(bytes memory raw, uint offset) internal pure returns(uint, bytes memory, uint64) {
        uint size = raw.toUint16(offset);
        offset = offset + 2;
        bytes memory extra = raw.slice(offset, size);
        offset = offset + size;
        uint64 timestamp = raw.toUint64(offset);
        offset = offset + 8;
        return (offset, extra, timestamp);
    }

    function parseEventAsset(bytes memory raw, uint offset) internal pure returns(uint, uint128, uint256) {
        uint128 id = raw.toUint128(offset);
        require(id > 0, "invalid asset");
        offset = offset + 16;
        uint size = raw.toUint16(offset);
        offset = offset + 2;
        require(size <= 32, "integer out of bounds");
        uint256 amount = new bytes(32 - size).concat(raw.slice(offset, size)).toUint256(0);
        offset = offset + size;
        return (offset, id, amount);
    }

    function parseEventUser(bytes memory raw, uint offset) internal returns (uint, address) {
        uint16 size = raw.toUint16(offset);
        size = 2 + size * 16 + 2;
        bytes memory members = raw.slice(offset, size);
        offset = offset + size;
        address user = getOrCreateUserContract(members);
        return (offset, user);
    }

    function parseEventInput(uint128 id, bytes memory extra) internal returns (address, bytes memory) {
        uint offset = 0;
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

    function getOrCreateAssetContract(uint128 id, string memory symbol, string memory name) internal returns (address) {
        address old = contracts[id];
        if (old != address(0)) {
            return old;
        }
        bytes memory code = getAssetContractCode(id, symbol, name);
        address asset = getContractAddress(code);
        if (assets[asset] > 0) {
            return asset;
        }
        address addr = deploy(code, VERSION);
        require(addr == asset, "malformed asset contract address");
        assets[asset] = id;
        contracts[id] = asset;
        addresses.push(asset);
        emit AssetCreated(asset, id);
        return asset;
    }

    function getOrCreateUserContract(bytes memory members) internal returns (address) {
        uint id = uint256(keccak256(members));
        address old = contracts[id];
        if (old != address(0)) {
            return old;
        }
        bytes memory code = getUserContractCode(members);
        address user = getContractAddress(code);
        if (users[user].length > 0) {
            return user;
        }
        address addr = deploy(code, VERSION);
        require(addr == user, "malformed user contract address");
        users[user] = members;
        contracts[id] = user;
        addresses.push(user);
        emit UserCreated(user, members);
        return user;
    }

    function getUserContractCode(bytes memory members) internal pure returns (bytes memory) {
        bytes memory code = type(User).creationCode;
        bytes memory args = abi.encode(members);
        return abi.encodePacked(code, args);
    }

    function getAssetContractCode(uint id, string memory symbol, string memory name) internal pure returns (bytes memory) {
        bytes memory code = type(Asset).creationCode;
        bytes memory args = abi.encode(id, name, symbol);
        return abi.encodePacked(code, args);
    }

    function getContractAddress(bytes memory code) internal view returns (address) {
        code = abi.encodePacked(bytes1(0xff), address(this), VERSION, keccak256(code));
        return address(uint160(uint(keccak256(code))));
    }

    function deploy(bytes memory bytecode, uint _salt) internal returns (address) {
        address addr;
        assembly {
            addr := create2(
                callvalue(),
                add(bytecode, 0x20),
                mload(bytecode),
                _salt
            )

            if iszero(extcodesize(addr)) {
                revert(0, 0)
            }
        }
        return addr;
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
