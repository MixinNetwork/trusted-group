// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {IERC20} from './erc20.sol';

// a simple vault contract
contract Vault {
  function deposit(address asset, uint256 amount) public {
    IERC20(asset).transferFrom(msg.sender, address(this), amount);
  }

  function withdraw(address asset, uint256 amount) public {
    IERC20(asset).transfer(msg.sender, amount);
  }

  function refund(address asset, uint256 amount) public {
    IERC20(asset).transferFrom(msg.sender, address(this), amount);
    IERC20(asset).transfer(msg.sender, amount);
  }
}
