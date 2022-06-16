// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {Bytes} from './libs/Bytes.sol';
import {IRegistry,Registrable} from './Registrable.sol';
import {IERC20} from './Asset.sol';

contract User is Registrable {
    using Bytes for bytes;

    event ProcessCalled(bytes input, bool result, bytes output);

    bytes public members;

    constructor(bytes memory _members) {
        members = _members;
    }

    function run(address asset, uint256 amount, bytes memory extra) external onlyRegistry() returns (bool) {
        if (extra.length < 28) {
            IRegistry(registry).claim(asset, amount);
            return true;
        }
        uint16 count = extra.toUint16(0);
        if (count < 1 || count > 16) {
            IRegistry(registry).claim(asset, amount);
            return true;
        }

        for (uint offset = 2; count >= 0; count--) {
            if (offset + 20 > extra.length) {
                break;
            }
            address process = extra.toAddress(offset);
            offset = offset + 20;
            IERC20(asset).approve(process, 0);
            IERC20(asset).approve(process, amount);

            if (offset + 2 > extra.length) {
                break;
            }
            uint size = extra.toUint16(offset);
            offset = offset + 2;

            if (offset + size > extra.length) {
                break;
            }
            bytes memory input = extra.slice(offset, size);
            (bool result, bytes memory output) = process.call(input);
            offset = offset + size;

            emit ProcessCalled(input, result, output);
            if (!result) {
                break;
            }
        }
        try IRegistry(registry).claim(asset, amount) {} catch {}
        return true;
    }
}
