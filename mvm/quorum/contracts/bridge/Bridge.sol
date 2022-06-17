// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

interface IERC20 {
    function transfer(address to, uint256 value) external returns (bool);

    function transferFrom(
        address from,
        address to,
        uint256 value
    ) external returns (bool);
}

contract Bridge {
    uint256 public constant BASE = 10000000000;

    address public immutable XIN;
    address public immutable OWNER;
    mapping(address => address) public bridges;

    event Crossed(
        address indexed asset,
        address indexed from,
        address indexed to,
        uint256 amount
    );

    constructor(address xin) {
        XIN = xin;
        OWNER = msg.sender;
    }

    receive() external payable {
        if (msg.sender == OWNER) {
            return;
        }

        address receiver = bridges[msg.sender];
        uint256 amount = msg.value / BASE;
        require(receiver != address(0), "no binding");
        require(amount > 0, "too small");

        IERC20(XIN).transfer(receiver, amount);
        emit Crossed(XIN, msg.sender, receiver, amount);
    }

    function bind(address receiver) public {
        require(receiver != address(0), "invalid address");
        bridges[msg.sender] = receiver;
        bridges[receiver] = msg.sender;
    }

    function deposit(address asset, uint256 amount) public {
        address receiver = bridges[msg.sender];
        require(receiver != address(0), "no binding");
        require(amount > 0, "too small");

        if (asset == XIN) {
            depositXIN(receiver, amount);
        } else {
            IERC20(asset).transferFrom(msg.sender, receiver, amount);
        }

        emit Crossed(XIN, msg.sender, receiver, amount);
    }

    function depositXIN(address receiver, uint256 amount) internal {
        IERC20(XIN).transferFrom(msg.sender, address(this), amount);
        payable(receiver).transfer(amount * BASE);
    }
}
