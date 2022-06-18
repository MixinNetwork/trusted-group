// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {IRegistry, Registrable} from "./Registrable.sol";

/**
 * @title ERC20 interface
 * @dev see https://github.com/ethereum/EIPs/issues/20
 */
interface IERC20 {
    function balanceOf(address who) external view returns (uint256);

    function transfer(address to, uint256 value) external returns (bool);

    function allowance(address owner, address spender)
        external
        view
        returns (uint256);

    function transferFrom(
        address from,
        address to,
        uint256 value
    ) external returns (bool);

    function approve(address spender, uint256 value) external returns (bool);

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(
        address indexed owner,
        address indexed spender,
        uint256 value
    );
}

/**
 * @title Standard ERC20 token
 *
 * @dev Implementation of the basic standard token.
 * @dev https://github.com/ethereum/EIPs/issues/20
 * @dev Based on code by FirstBlood: https://github.com/Firstbloodio/token/blob/master/smart_contract/FirstBloodToken.sol
 */
abstract contract StandardToken is IERC20 {
    mapping(address => uint256) balances;
    mapping(address => mapping(address => uint256)) allowed;

    /**
     * @dev Gets the balance of the specified address.
     * @param _owner The address to query the the balance of.
     * @return balance representing the amount owned by the passed address.
     */
    function balanceOf(address _owner)
        public
        view
        override
        returns (uint256 balance)
    {
        return balances[_owner];
    }

    /**
     * @dev transfer token for a specified address
     * @param _to The address to transfer to.
     * @param _value The amount to be transferred.
     */
    function _transfer(
        address _from,
        address _to,
        uint256 _value
    ) internal {
        balances[_from] = balances[_from] - _value;
        balances[_to] = balances[_to] + _value;
        emit Transfer(_from, _to, _value);
    }

    /**
     * @dev Transfer tokens from one address to another
     * @param _from address The address which you want to send tokens from
     * @param _to address The address which you want to transfer to
     * @param _value uint256 the amount of tokens to be transferred
     */
    function _transferFrom(
        address _from,
        address _to,
        uint256 _value
    ) internal {
        uint256 _allowance = allowed[_from][msg.sender];
        allowed[_from][msg.sender] = _allowance - _value;
        _transfer(_from, _to, _value);
    }

    /**
     * @dev Approve the passed address to spend the specified amount of tokens on behalf of msg.sender.
     * @param _spender The address which will spend the funds.
     * @param _value The amount of tokens to be spent.
     */
    function approve(address _spender, uint256 _value)
        public
        override
        returns (bool)
    {
        // To change the approve amount you first have to reduce the addresses`
        //  allowance to zero by calling `approve(_spender, 0)` if it is not
        //  already 0 to mitigate the race condition described here:
        //  https://github.com/ethereum/EIPs/issues/20#issuecomment-263524729
        require(
            (_value == 0) || (allowed[msg.sender][_spender] == 0),
            "approve on a non-zero allowance"
        );
        allowed[msg.sender][_spender] = _value;
        emit Approval(msg.sender, _spender, _value);
        return true;
    }

    /**
     * @dev Function to check the amount of tokens that an owner allowed to a spender.
     * @param _owner address The address which owns the funds.
     * @param _spender address The address which will spend the funds.
     * @return remaining uint256 specifying the amount of tokens still available for the spender.
     */
    function allowance(address _owner, address _spender)
        public
        view
        override
        returns (uint256 remaining)
    {
        return allowed[_owner][_spender];
    }
}

contract Asset is Registrable, StandardToken {
    uint128 public immutable id;

    string public name;
    string public symbol;
    uint256 public totalSupply;
    uint8 public constant decimals = 8;

    constructor(
        uint128 _id,
        string memory _name,
        string memory _symbol
    ) {
        id = _id;
        name = _name;
        symbol = _symbol;
    }

    function transferWithExtra(
        address to,
        uint256 value,
        bytes memory extra
    ) public returns (bool) {
        _transfer(msg.sender, to, value);
        IRegistry(registry).burn(to, value, extra);
        return true;
    }

    function transfer(address to, uint256 value)
        public
        override
        returns (bool)
    {
        return transferWithExtra(to, value, new bytes(0));
    }

    function transferFrom(
        address from,
        address to,
        uint256 value
    ) public override returns (bool) {
        _transferFrom(from, to, value);
        IRegistry(registry).burn(to, value, new bytes(0));
        return true;
    }

    function mint(address to, uint256 value) external onlyRegistry {
        balances[to] = balances[to] + value;
        totalSupply = totalSupply + value;
        emit Transfer(registry, to, value);
    }

    function burn(address to, uint256 value) external onlyRegistry {
        balances[to] = balances[to] - value;
        totalSupply = totalSupply - value;
        emit Transfer(to, registry, value);
    }
}
