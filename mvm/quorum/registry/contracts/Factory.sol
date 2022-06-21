// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {Asset} from "./Asset.sol";
import {User} from "./User.sol";

abstract contract Factory {
    uint256 public constant VERSION = 1;

    event UserCreated(address indexed at, bytes members);
    event AssetCreated(address indexed at, uint256 id);

    mapping(address => bytes) public users;
    mapping(address => uint128) public assets;
    mapping(uint256 => address) public contracts;

    function getOrCreateAssetContract(
        uint128 id,
        string memory symbol,
        string memory name
    ) internal returns (address) {
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
        emit AssetCreated(asset, id);
        return asset;
    }

    function getOrCreateUserContract(bytes memory members)
        internal
        returns (address)
    {
        uint256 id = uint256(keccak256(members));
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
        emit UserCreated(user, members);
        return user;
    }

    function getUserContractCode(bytes memory members)
        internal
        pure
        returns (bytes memory)
    {
        bytes memory code = type(User).creationCode;
        bytes memory args = abi.encode(members);
        return abi.encodePacked(code, args);
    }

    function getAssetContractCode(
        uint256 id,
        string memory symbol,
        string memory name
    ) internal pure returns (bytes memory) {
        bytes memory code = type(Asset).creationCode;
        bytes memory args = abi.encode(id, name, symbol);
        return abi.encodePacked(code, args);
    }

    function getContractAddress(bytes memory code)
        internal
        view
        returns (address)
    {
        code = abi.encodePacked(
            bytes1(0xff),
            address(this),
            VERSION,
            keccak256(code)
        );
        return address(uint160(uint256(keccak256(code))));
    }

    function deploy(bytes memory bytecode, uint256 _salt)
        internal
        returns (address)
    {
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
