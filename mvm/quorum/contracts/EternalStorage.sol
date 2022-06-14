// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;
import "hardhat/console.sol";

contract EternalStorage {

    mapping(bytes32 => uint) UIntStorage;

    function getUIntValue(bytes32 record) public view returns (uint){
        return UIntStorage[record];
    }

    function setUIntValue(bytes32 record, uint value) public
    {
        UIntStorage[record] = value;
    }

    mapping(bytes32 => string) StringStorage;

    function getStringValue(bytes32 record) public view returns (string memory){
        return StringStorage[record];
    }

    function setStringValue(bytes32 record, string calldata value) public
    {
        require(keccak256(abi.encodePacked(value)) == record, "invalid record or value");
        StringStorage[record] = value;
    }

    mapping(bytes32 => address) AddressStorage;

    function getAddressValue(bytes32 record) public view returns (address){
        return AddressStorage[record];
    }

    function setAddressValue(bytes32 record, address value) public
    {
        AddressStorage[record] = value;
    }

    mapping(bytes32 => bytes) BytesStorage;

    function getBytesValue(bytes32 record) public view returns (bytes memory){
        return BytesStorage[record];
    }

    function setBytesValue(bytes32 record, bytes calldata value) public
    {
        require(keccak256(value) == record, "invalid record or value");
        BytesStorage[record] = value;
    }

    mapping(bytes32 => bool) BooleanStorage;

    function getBooleanValue(bytes32 record) public view returns (bool){
        return BooleanStorage[record];
    }

    function setBooleanValue(bytes32 record, bool value) public
    {
        BooleanStorage[record] = value;
    }
    
    mapping(bytes32 => int) IntStorage;

    function getIntValue(bytes32 record) public view returns (int){
        return IntStorage[record];
    }

    function setIntValue(bytes32 record, int value) public
    {
        IntStorage[record] = value;
    }
}
