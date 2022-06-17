// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

contract Faucet {
    address public immutable OWNER;
    mapping(address => uint256) public claims;

    event Distribution(address indexed receiver, uint256 amount);

    constructor() {
        OWNER = msg.sender;
    }

    receive() external payable {}

    function distribute(address receiver, uint256 total) public {
        require(msg.sender == OWNER);
        uint256 amount = total - claims[receiver];
        payable(receiver).transfer(amount);
        claims[receiver] = total;
        emit Distribution(receiver, amount);
    }
}