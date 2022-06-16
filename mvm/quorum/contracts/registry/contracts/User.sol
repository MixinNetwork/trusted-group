// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {Bytes} from './libs/Bytes.sol';
import {IRegistry,Registrable} from './Registrable.sol';
import {IERC20} from './Asset.sol';

contract User is Registrable {
    using Bytes for bytes;

    bytes public members;

    constructor(bytes memory _members) {
        members = _members;
    }

    function run(address asset, uint256 amount, bytes memory extra) external onlyRegistry() returns (bool result) {
        if (extra.length < 24) {
            return true;
        }
        address process = extra.toAddress(0);
        IERC20(asset).approve(process, 0);
        IERC20(asset).approve(process, amount);
        bytes memory input = extra.slice(20, extra.length - 20);
        (result, input) = process.call(input);
        try IRegistry(registry).claim(asset, amount) {} catch {}
        return result;
    }
}
