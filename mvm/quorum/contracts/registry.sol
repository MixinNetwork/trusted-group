// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {BytesLib} from './bytes.sol';
import {BLS} from './bls.sol';


/**
    * @title SafeMath
* @dev Math operations with safety checks that throw on error
*/
library SafeMath {
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a * b;
        assert(a == 0 || c / a == b);
        return c;
    }

    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        // assert(b > 0); // Solidity automatically throws when dividing by 0
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold
        return c;
    }

    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        assert(b <= a);
        return a - b;
    }

    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        assert(c >= a);
        return c;
    }
}

/**
    * @title ERC20 interface
* @dev see https://github.com/ethereum/EIPs/issues/20
    */
interface ERC20 {
    function balanceOf(address who) external view returns (uint256);
    function transfer(address to, uint256 value) external returns (bool);
    function allowance(address owner, address spender) external view returns (uint256);
    function transferFrom(address from, address to, uint256 value) external returns (bool);
    function approve(address spender, uint256 value) external returns (bool);
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
}

/**
    * @title Standard ERC20 token
*
    * @dev Implementation of the basic standard token.
    * @dev https://github.com/ethereum/EIPs/issues/20
    * @dev Based on code by FirstBlood: https://github.com/Firstbloodio/token/blob/master/smart_contract/FirstBloodToken.sol
    */
abstract contract StandardToken is ERC20 {
    using SafeMath for uint256;

    mapping(address => uint256) balances;
    mapping (address => mapping (address => uint256)) allowed;

    /**
        * @dev Gets the balance of the specified address.
        * @param _owner The address to query the the balance of.
        * @return balance representing the amount owned by the passed address.
        */
    function balanceOf(address _owner) public view override returns (uint256 balance) {
        return balances[_owner];
    }

    /**
        * @dev transfer token for a specified address
    * @param _to The address to transfer to.
        * @param _value The amount to be transferred.
        */
    function _transfer(address _from, address _to, uint256 _value) public returns (bool) {
        require(_to != address(0));

        // SafeMath.sub will throw if there is not enough balance.
        balances[_from] = balances[_from].sub(_value);
        balances[_to] = balances[_to].add(_value);
        emit Transfer(_from, _to, _value);
        return true;
    }

    /**
        * @dev Transfer tokens from one address to another
    * @param _from address The address which you want to send tokens from
    * @param _to address The address which you want to transfer to
    * @param _value uint256 the amount of tokens to be transferred
    */
    function transferFrom(address _from, address _to, uint256 _value) public override returns (bool) {
        uint256 _allowance = allowed[_from][msg.sender];
        require(_to != address(0));
        require (_value <= _allowance);
        allowed[_from][msg.sender] = _allowance.sub(_value);
        return _transfer(_from, _to, _value);
    }

    /**
        * @dev Approve the passed address to spend the specified amount of tokens on behalf of msg.sender.
        * @param _spender The address which will spend the funds.
        * @param _value The amount of tokens to be spent.
        */
    function approve(address _spender, uint256 _value) public override returns (bool) {
        // To change the approve amount you first have to reduce the addresses`
        //  allowance to zero by calling `approve(_spender, 0)` if it is not
        //  already 0 to mitigate the race condition described here:
        //  https://github.com/ethereum/EIPs/issues/20#issuecomment-263524729
        require((_value == 0) || (allowed[msg.sender][_spender] == 0));
        allowed[msg.sender][_spender] = _value;
        emit Approval(msg.sender, _spender, _value);
        return true;
    }

    /**
        * @dev Function to check the amount of tokens that an owner allowed to a spender.
        * @param _owner address The address which owns the funds.
        * @param _spender address The address which will spend the funds.
        * @return remaining uint256 specifying the amount of tokens still available for the spender.
        */
    function allowance(address _owner, address _spender) public view override returns (uint256 remaining) {
        return allowed[_owner][_spender];
    }
}

contract MixinUser {
    address public registry;
    bytes public members;

    modifier onlyRegistry() {
        require(msg.sender == registry, "not registry");
        _;
    }

    constructor(bytes memory _members) onlyRegistry() {
        registry = msg.sender;
        members = _members;
    }

    function run(address process, bytes memory input) external onlyRegistry() returns (bool, bytes memory) {
        return process.call(input);
    }
}

contract MixinAsset is StandardToken {
    address public registry;
    uint public id;

    string public name;
    string public symbol;
    uint8 public constant decimals = 8;
    uint256 public totalSupply;

    modifier onlyRegistry() {
        require(msg.sender == registry, "not registry");
        _;
    }

    constructor(uint _id) onlyRegistry() {
        registry = msg.sender;
        id = _id;
    }

    function transfer(address _to, uint256 _value) public override returns (bool) {
        _transfer(msg.sender, _to, _value);
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

    event UserCreated(bytes members, address at);
    event AssetCreated(uint id, address at);
    event MixinTransaction(bytes);

    uint256[4] public GROUP;
    uint256 public NONCE = 0;
    mapping(address => bytes) users;
    mapping(address => uint) assets;

    struct Event {
        uint64 nonce;
        MixinAsset asset;
        uint256 amount;
        address process;
        bytes input;
        uint64 timestamp;
        MixinUser user;
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

        uint256 size = 0;
        uint256 offset = 0;
        evt.process = raw.toAddress(offset);
        offset = offset + 20;

        evt.nonce = raw.toUint64(offset);
        require(evt.nonce == NONCE, "invalid nonce");
        NONCE = NONCE + 1;
        offset = offset + 8;

        uint128 id = raw.toUint128(offset);
        require(id > 0, "invalid asset");
        offset = offset + 16;
        evt.asset = MixinAsset(getOrCreateEventAsset(id));

        size = raw.toUint16(offset);
        offset = offset + 2;
        require(size <= 32, "integer out of bounds");
        evt.amount = new bytes(32 - size).concat(raw.slice(offset, size)).toUint256(0);
        offset = offset + size;

        size = raw.toUint16(offset);
        offset = offset + 2;
        evt.input = raw.slice(offset, size);
        offset = offset + size;

        evt.timestamp = raw.toUint64(offset);
        offset = offset + 8;

        size = raw.toUint16(offset);
        size = 2 + size * 16 + 2;
        bytes memory members = raw.slice(offset, size);
        offset = offset + size;
        evt.user = MixinUser(getOrCreateEventUser(members));

        offset = offset + 2;
        evt.sig = [raw.toUint256(offset), raw.toUint256(offset+32)];
        uint256[2] memory message = raw.slice(0, offset-2).concat(new bytes(2)).hashToPoint();
        require(evt.sig.verifySingle(GROUP, message), "invalid signature");

        offset = offset + 64;
        require(raw.length == offset, "malformed event encoding");

        evt.asset.mint(address(evt.user), evt.amount);
        return evt.user.run(evt.process, evt.input);
    }

    function getOrCreateEventAsset(uint id) public returns (address) {
        address asset = getAssetAddress(id);
        if (assets[asset] > 0) {
            return asset;
        }
        bytes memory code = getAssetContractCode(id);
        address addr = deploy(code, id);
        require(addr == asset, "malformed user contract address");
        assets[asset] = id;
        emit AssetCreated(id, asset);
        return asset;
    }

    function getOrCreateEventUser(bytes memory members) public returns (address) {
        address user = getUserAddress(members);
        if (users[user].length > 0) {
            return user;
        }

        bytes memory code = getUserContractCode(members);
        uint salt = uint(keccak256(members));
        address addr = deploy(code, salt);
        require(addr == user, "malformed user contract address");
        users[user] = members;
        emit UserCreated(members, user);
        return user;
    }

    function getUserContractCode(bytes memory members) public pure returns (bytes memory) {
        bytes memory code = type(MixinUser).creationCode;
        bytes memory args = abi.encode(members);
        return abi.encodePacked(code, args);
    }

    function getUserAddress(bytes memory members) public view returns (address) {
        uint salt = uint(keccak256(members));
        bytes memory code = getUserContractCode(members);
        code = abi.encodePacked(bytes1(0xff), address(this), salt, keccak256(code));
        return address(uint160(uint(keccak256(code))));
    }

    function getAssetContractCode(uint id) public pure returns (bytes memory) {
        bytes memory code = type(MixinAsset).creationCode;
        bytes memory args = abi.encode(id);
        return abi.encodePacked(code, args);
    }

    function getAssetAddress(uint id) public view returns (address) {
        bytes memory code = getAssetContractCode(id);
        code = abi.encodePacked(bytes1(0xff), address(this), id, keccak256(code));
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
