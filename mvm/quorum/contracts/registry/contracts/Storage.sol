// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

contract Storage {
  mapping(uint256 => bytes) internal values;

  function read(uint256 _key) public view returns (bytes memory) {
    return values[_key];
  }

  function write(uint256 _key, bytes memory raw) public {
    uint key = uint256(keccak256(raw));
    require(key == _key, "invalid key or raw");
    values[_key] = raw;
  }
}
