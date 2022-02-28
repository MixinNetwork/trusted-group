// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {IERC20} from './erc20.sol';

contract Bridge {
  mapping(address => address) public bridges;

  function bind(address addr) public {
    require(addr != address(0), "invalid address");
    bridges[msg.sender] = addr;
  }

  function deposit(address asset, uint256 amount) public {
    IERC20(asset).transferFrom(msg.sender, bridges[msg.sender], amount);
  }
}
