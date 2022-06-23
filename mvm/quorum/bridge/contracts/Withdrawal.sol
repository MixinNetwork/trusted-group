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

contract Withdrawal {
    uint256 public constant BASE = 10000000000;

    address public immutable BRIDGE;
    address public immutable XIN;

    constructor(address bridge, address xin) {
        BRIDGE = bridge;
        XIN = xin;
    }

    function submit(
        address receiver,
        address asset,
        uint256 amount,
        address feeAsset,
        uint256 feeAmount,
        bytes memory ma,
        bytes memory mb
    ) public payable {
        require(feeAsset != XIN, "invalid fee asset");
        if (asset == XIN) {
            require(msg.value / BASE == amount, "invalid withdrawal amount");
            transferXIN(receiver, ma);
        } else {
            transferERC20(receiver, asset, amount, ma);
        }
        transferERC20(receiver, feeAsset, feeAmount, mb);
    }

    function transferXIN(address receiver, bytes memory input) internal {
        bytes memory data = abi.encodeWithSignature(
            "release(address,bytes)",
            receiver,
            input
        );
        (bool success, bytes memory result) = BRIDGE.call{value: msg.value}(
            data
        );
        require(success, string(result));
    }

    function transferERC20(
        address receiver,
        address asset,
        uint256 amount,
        bytes memory input
    ) internal {
        IERC20(asset).transferFrom(msg.sender, address(this), amount);
        IERC20(asset).transferWithExtra(receiver, amount, input);
    }
}
