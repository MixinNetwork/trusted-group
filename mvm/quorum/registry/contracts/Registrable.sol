// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

interface IRegistry {
    function claim(address asset, uint256 amount) external returns (bool);

    function burn(address user, uint256 amount, bytes memory extra) external returns (bool);
}

abstract contract Registrable {
    address public registry;

    modifier onlyRegistry() {
        require(msg.sender == registry, "not registry");
        _;
    }

    constructor() {
        registry = msg.sender;
    }
}
