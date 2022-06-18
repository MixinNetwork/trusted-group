// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

interface IERC20 {
    function transferWithExtra(
        address to,
        uint256 value,
        bytes memory extra
    ) external returns (bool);

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

    event Vault(address indexed from, uint256 amount);
    event Bound(address indexed from, address indexed to);
    event Through(
        address indexed asset,
        address indexed from,
        address indexed to,
        uint256 amount
    );

    constructor(address xin) {
        XIN = xin;
        OWNER = msg.sender;
    }

    fallback(bytes calldata input) external payable returns (bytes memory) {
        uint256 amount = msg.value / BASE;
        require(amount > 0, "too small");

        if (msg.sender == OWNER) {
            emit Vault(msg.sender, amount);
            return new bytes(0);
        }

        address receiver = bridges[msg.sender];
        require(receiver != address(0), "no binding");

        IERC20(XIN).transferWithExtra(receiver, amount, input);
        emit Through(XIN, msg.sender, receiver, amount);
        return new bytes(0);
    }

    function vault(address asset, uint256 amount) public {
        require(asset == XIN, "only XIN");
        IERC20(asset).transferFrom(msg.sender, address(this), amount);
        emit Vault(msg.sender, amount);
    }

    function bind(address receiver) public {
        require(receiver != address(0), "invalid address");
        bridges[msg.sender] = receiver;
        emit Bound(msg.sender, receiver);
    }

    function pass(address asset, uint256 amount) public {
        address receiver = bridges[msg.sender];
        require(receiver != address(0), "no binding");
        require(amount > 0, "too small");

        if (asset == XIN) {
            passXIN(receiver, amount);
        } else {
            IERC20(asset).transferFrom(msg.sender, receiver, amount);
        }

        emit Through(XIN, msg.sender, receiver, amount);
    }

    function passXIN(address receiver, uint256 amount) internal {
        IERC20(XIN).transferFrom(msg.sender, address(this), amount);
        payable(receiver).transfer(amount * BASE);
    }
}
