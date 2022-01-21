// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {BytesLib} from './bytes.sol';
import {BLS} from './bls.sol';
import {StandardToken} from './erc20.sol';

contract Registrable {
    address public registry;

    modifier onlyRegistry() {
        require(msg.sender == registry, "not registry");
        _;
    }
}

contract MixinUser is Registrable {
    bytes public members;

    constructor(bytes memory _members) {
        registry = msg.sender;
        members = _members;
    }

    function run(address process, bytes memory input) external onlyRegistry() returns (bool, bytes memory) {
        return process.call(input);
    }
}

contract MixinAsset is Registrable, StandardToken {
    uint public id;

    string public name;
    string public symbol;
    uint256 public totalSupply;
    uint8 public constant decimals = 8;

    constructor(uint _id, string memory _name, string memory _symbol) {
        registry = msg.sender;
        id = _id;
        name = _name;
        symbol = _symbol;
    }

    function transfer(address _to, uint256 _value) public override returns (bool) {
        _transfer(msg.sender, _to, _value);
        Registry(registry).burn(_to, _value);
        return true;
    }

    function transferFrom(address _from, address _to, uint256 _value) public override returns (bool) {
        _transferFrom(_from, _to, _value);
        Registry(registry).burn(_to, _value);
        return true;
    }

    function mint(address to, uint256 amount) external onlyRegistry() {
        balances[to] = balances[to] + amount;
        totalSupply = totalSupply + amount;
        emit Transfer(registry, to, amount);
    }

    function burn(address _to, uint256 _value) external onlyRegistry() {
        balances[_to] = balances[_to] - _value;
        totalSupply = totalSupply - _value;
        emit Transfer(_to, registry, _value);
    }
}

contract Registry {
    using BytesLib for bytes;
    using BLS for uint256[2];
    using BLS for bytes;

    event UserCreated(address at, bytes members);
    event AssetCreated(address at, uint id);
    event MixinTransaction(bytes);

    uint256[4] public GROUP;
    uint256 public NONCE = 0;
    uint256 public constant VERSION = 1;
    mapping(address => bytes) users;
    mapping(address => uint) assets;

    struct Event {
        uint64 nonce;
        address user;
        address asset;
        uint256 amount;
        address process;
        bytes input;
        uint64 timestamp;
        uint256[2] sig;
    }

    constructor(uint256[4] memory _group) {
        GROUP = _group;
    }

    function claim(address asset, uint256 amount) external returns (bool) {
        if (users[msg.sender].length == 0) {
            return false;
        }
        require(assets[asset] > 0);
        MixinAsset(msg.sender).burn(msg.sender, amount);
        // emit MixinTransaction
        return true;
    }

    function burn(address user, uint256 amount) external returns (bool) {
        if (users[user].length == 0) {
            return true;
        }
        require(assets[msg.sender] > 0);
        MixinAsset(msg.sender).burn(user, amount);
        // emit MixinTransaction
        return true;
    }

    function mixin(bytes memory raw) public returns (bool, bytes memory) {
        require(raw.length >= 141, "event data too small");

        Event memory evt;
        uint256 offset = 0;

        evt.nonce = raw.toUint64(offset);
        require(evt.nonce == NONCE, "invalid nonce");
        NONCE = NONCE + 1;
        offset = offset + 8;

        (offset, evt.user) = parseEventUser(raw, offset);
        (offset, evt.asset) = parseEventAsset(raw, offset);
        (offset, evt.amount) = parseEventAmount(raw, offset);
        (offset, evt.process, evt.input) = parseEventInput(raw, offset);

        evt.timestamp = raw.toUint64(offset);
        offset = offset + 8;

        offset = offset + 2;
        evt.sig = [raw.toUint256(offset), raw.toUint256(offset+32)];
        uint256[2] memory message = raw.slice(0, offset-2).concat(new bytes(2)).hashToPoint();
        require(evt.sig.verifySingle(GROUP, message), "invalid signature");

        offset = offset + 64;
        require(raw.length == offset, "malformed event encoding");

        MixinAsset(evt.asset).mint(evt.user, evt.amount);
        return MixinUser(evt.user).run(evt.process, evt.input);
    }

    function parseEventInput(bytes memory raw, uint offset) public pure returns(uint, address, bytes memory) {
        address process = raw.toAddress(offset);
        offset = offset + 20;

        uint size = raw.toUint16(offset);
        offset = offset + 2;
        bytes memory input = raw.slice(offset, size);
        offset = offset + size;
        return (offset, process, input);
    }

    function parseEventAmount(bytes memory raw, uint offset) public pure returns(uint, uint256) {
        uint size = raw.toUint16(offset);
        offset = offset + 2;
        require(size <= 32, "integer out of bounds");
        uint256 amount = new bytes(32 - size).concat(raw.slice(offset, size)).toUint256(0);
        offset = offset + size;
        return (offset, amount);
    }

    function parseEventUser(bytes memory raw, uint offset) public returns (uint, address) {
        uint16 size = raw.toUint16(offset);
        size = 2 + size * 16 + 2;
        bytes memory members = raw.slice(offset, size);
        offset = offset + size;
        return (offset, getOrCreateEventUser(members));
    }

    function parseEventAsset(bytes memory raw, uint offset) public returns (uint, address) {
        uint128 id = raw.toUint128(offset);
        require(id > 0, "invalid asset");
        offset = offset + 16;
        uint16 size = raw.toUint16(offset);
        offset = offset + 2;
        string memory symbol = string(raw.slice(offset, size));
        offset = offset + size;
        size = raw.toUint16(offset);
        offset = offset + 2;
        string memory name = string(raw.slice(offset, size));
        offset = offset + size;
        address addr = getOrCreateEventAsset(id, symbol, name);
        return (offset, addr);
    }

    function getOrCreateEventAsset(uint id, string memory symbol, string memory name) public returns (address) {
        bytes memory code = getAssetContractCode(id, symbol, name);
        address asset = getContractAddress(code);
        if (assets[asset] > 0) {
            return asset;
        }
        address addr = deploy(code, id);
        require(addr == asset, "malformed user contract address");
        assets[asset] = id;
        emit AssetCreated(asset, id);
        return asset;
    }

    function getOrCreateEventUser(bytes memory members) public returns (address) {
        bytes memory code = getUserContractCode(members);
        address user = getContractAddress(code);
        if (users[user].length > 0) {
            return user;
        }
        uint salt = uint(keccak256(members));
        address addr = deploy(code, salt);
        require(addr == user, "malformed user contract address");
        users[user] = members;
        emit UserCreated(user, members);
        return user;
    }

    function getUserContractCode(bytes memory members) public pure returns (bytes memory) {
        bytes memory code = type(MixinUser).creationCode;
        bytes memory args = abi.encode(members);
        return abi.encodePacked(code, args);
    }

    function getAssetContractCode(uint id, string memory symbol, string memory name) public pure returns (bytes memory) {
        bytes memory code = type(MixinAsset).creationCode;
        bytes memory args = abi.encode(id, symbol, name);
        return abi.encodePacked(code, args);
    }

    function getContractAddress(bytes memory code) public view returns (address) {
        code = abi.encodePacked(bytes1(0xff), address(this), VERSION, keccak256(code));
        return address(uint160(uint(keccak256(code))));
    }

    function deploy(bytes memory bytecode, uint _salt) public returns (address) {
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
}
