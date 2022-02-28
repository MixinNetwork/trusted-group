// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.4 <0.9.0;

import {IERC20} from './erc20.sol';

// a simple vault contract
contract Vault {
  mapping(address => mapping(address => uint256)) public balances;

  function deposit(address asset, uint256 amount) public {
    IERC20(asset).transferFrom(msg.sender, address(this), amount);
    balances[asset][msg.sender] += amount;
  }

  function withdraw(address asset, uint256 amount) public {
    require(balances[asset][msg.sender] > amount);
    balances[asset][msg.sender] -= amount;
    IERC20(asset).transfer(msg.sender, amount);
  }

  function refund(address asset, uint256 amount) public {
    IERC20(asset).transferFrom(msg.sender, address(this), amount);
    balances[asset][msg.sender] += amount;
    balances[asset][msg.sender] -= amount / 7;
    IERC20(asset).transfer(msg.sender, amount / 7);
  }
}
