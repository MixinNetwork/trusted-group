// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {Bytes} from "./libs/Bytes.sol";
import {IRegistry, Registrable} from "./Registrable.sol";
import {IERC20} from "./Asset.sol";

contract User is Registrable {
    using Bytes for bytes;

    event ProcessCalled(
        address indexed process,
        bytes input,
        bool result,
        bytes output
    );

    bytes public members;

    constructor(bytes memory _members) {
        members = _members;
    }

    function run(
        address asset,
        uint256 amount,
        bytes memory extra
    ) external onlyRegistry returns (bool) {
        if (extra.length < 28) {
            IRegistry(registry).claim(asset, amount);
            return true;
        }
        uint16 count = extra.toUint16(0);
        if (count < 1 || count > 16) {
            IRegistry(registry).claim(asset, amount);
            return true;
        }

        for (uint256 offset = 2; count >= 0 && offset < extra.length; count--) {
            bool primary = offset == 2;
            bytes memory data = extra.slice(offset, extra.length - offset);
            (uint256 size, bool success) = handle(data, asset, amount, primary);
            if (!success) {
                break;
            }
            offset = offset + size;
        }
        try IRegistry(registry).claim(asset, amount) {} catch {}
        return true;
    }

    function handle(
        bytes memory extra,
        address asset,
        uint256 amount,
        bool primary
    ) internal returns (uint256, bool) {
        uint256 offset = 0;
        if (offset + 20 > extra.length) {
            return (offset, false);
        }
        address process = extra.toAddress(offset);
        offset = offset + 20;
        if (primary) {
            IERC20(asset).approve(process, 0);
            IERC20(asset).approve(process, amount);
        }

        if (offset + 2 > extra.length) {
            return (offset, false);
        }
        uint256 size = extra.toUint16(offset);
        offset = offset + 2;

        if (offset + size > extra.length) {
            return (offset, false);
        }
        bytes memory input = extra.slice(offset, size);
        (bool result, bytes memory output) = process.call(input);
        offset = offset + size;

        emit ProcessCalled(process, input, result, output);
        return (offset, result);
    }
}
