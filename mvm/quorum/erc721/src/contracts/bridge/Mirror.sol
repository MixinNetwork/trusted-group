// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.9 <0.9.0;

import {Collectible} from "./Collectible.sol";

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

    function name() external returns (string memory);

    function symbol() external returns (string memory);
}

interface Factory {
    function contracts(uint256 id) external view returns (address);
}

contract Mirror {
    uint256 public constant VERSION = 1;
    uint256 internal constant AMOUNT = 100000000;
    bytes4 internal constant MAGIC_ON_ERC721_RECEIVED = 0x150b7a02;

    struct Token {
        address collection;
        uint256 id;
    }

    address public immutable FACTORY;
    mapping(address => address) public bridges;

    mapping(address => uint256) public collections;
    mapping(uint256 => address) public contracts;

    mapping(address => mapping(uint256 => address)) public tokens;
    mapping(address => Token) public mints;

    event CollectionCreated(address indexed at, uint256 id);
    event Bound(address indexed from, address indexed to);
    event Through(
        address indexed collection,
        address indexed from,
        address indexed to,
        uint256 id
    );

    constructor(address factory) {
        FACTORY = factory;
    }

    function onERC721Received(
        address,
        address _from,
        uint256 _tokenId,
        bytes memory _data
    ) external returns (bytes4) {
        address receiver = bridges[_from];
        if (receiver == address(0)) {
            (receiver, _data) = parseDataAsReceiver(_data);
        }
        require(receiver != address(0), "no address bound");

        Collectible(msg.sender).burn(_tokenId);
        address asset = tokens[msg.sender][_tokenId];
        IERC20(asset).transferWithExtra(receiver, AMOUNT, _data);
        emit Through(msg.sender, _from, receiver, _tokenId);
        return MAGIC_ON_ERC721_RECEIVED;
    }

    function bind(address receiver) public {
        require(receiver != address(0), "invalid address");
        bridges[msg.sender] = receiver;
        emit Bound(msg.sender, receiver);
    }

    function pass(address asset) public {
        address receiver = bridges[msg.sender];
        require(receiver != address(0), "no address bound");

        asset = canonical(asset);
        IERC20 erc20 = IERC20(asset);
        (bytes memory csb, uint256 collection) = parseName(bytes(erc20.name()));
        (bytes memory tsb, uint256 token) = parseSymbol(bytes(erc20.symbol()));
        address collectible = getOrCreateCollectibleContract(collection);

        bytes memory uri = "https://bridge.mvm.dev/collectibles/";
        uri = bytes.concat(uri, csb, "/", tsb, ".json");
        Collectible(collectible).mint(receiver, token, string(uri));
        tokens[collectible][token] = asset;
        mints[asset].collection = collectible;
        mints[asset].id = token;

        erc20.transferFrom(msg.sender, address(this), AMOUNT);
        emit Through(collectible, msg.sender, receiver, token);
    }

    function canonical(address asset) internal view returns (address) {
        uint256 id = uint256(uint160(asset));
        address another = Factory(FACTORY).contracts(id);
        if (another != address(0)) {
            return another;
        }
        return asset;
    }

    function getOrCreateCollectibleContract(uint256 collection)
        internal
        returns (address)
    {
        address old = contracts[collection];
        if (old != address(0)) {
            return old;
        }
        bytes memory code = getCollectibleContractCode(
            collection,
            "NFT",
            "Collectible"
        );
        address collectible = getContractAddress(code);
        if (collections[collectible] > 0) {
            return collectible;
        }
        address addr = deploy(code, VERSION);
        require(addr == collectible, "malformed collectible contract address");
        collections[collectible] = collection;
        contracts[collection] = collectible;
        emit CollectionCreated(collectible, collection);
        return collectible;
    }

    function getCollectibleContractCode(
        uint256 id,
        string memory symbol,
        string memory name
    ) internal pure returns (bytes memory) {
        bytes memory code = type(Collectible).creationCode;
        bytes memory args = abi.encode(id, name, symbol);
        return abi.encodePacked(code, args);
    }

    function getContractAddress(bytes memory code)
        internal
        view
        returns (address)
    {
        code = abi.encodePacked(
            bytes1(0xff),
            address(this),
            VERSION,
            keccak256(code)
        );
        return address(uint160(uint256(keccak256(code))));
    }

    function deploy(bytes memory bytecode, uint256 _salt)
        internal
        returns (address)
    {
        address addr;
        assembly {
            addr := create2(
                callvalue(),
                add(bytecode, 0x20),
                mload(bytecode),
                _salt
            )

            if iszero(extcodesize(addr)) {
                revert(0, 0)
            }
        }
        return addr;
    }

    function parseDataAsReceiver(bytes memory _bytes)
        internal
        pure
        returns (address, bytes memory)
    {
        require(_bytes.length >= 20, "toAddress_outOfBounds");
        address tempAddress;

        assembly {
            tempAddress := div(
                mload(add(add(_bytes, 0x20), 0)),
                0x1000000000000000000000000
            )
        }

        return (tempAddress, slice(_bytes, 20, _bytes.length - 20));
    }

    function parseName(bytes memory _bytes)
        internal
        pure
        returns (bytes memory, uint256)
    {
        require(
            _bytes.length > 12 && _bytes.length <= 44,
            "invalid collectible asset name length"
        );
        require(
            keccak256(slice(_bytes, 0, 12)) == keccak256(bytes("Collectible ")),
            "invalid collectible asset name prefix"
        );

        bytes memory name = slice(_bytes, 12, _bytes.length - 12);
        return (name, uint256(keccak256(name)));
    }

    function parseSymbol(bytes memory b)
        internal
        pure
        returns (bytes memory, uint256)
    {
        require(b.length > 4, "invalid collectible asset symbol length");
        require(
            keccak256(slice(b, 0, 4)) == keccak256(bytes("NFT#")),
            "invalid collectible asset symbol prefix"
        );
        uint256 result = 0;
        for (uint i = 4; i < b.length; i++) {
            uint256 c = uint256(uint8(b[i]));
            require(c >= 48 && c <= 57, "invalid collectible asset symbol id");
            result = result * 10 + (c - 48);
        }
        return (slice(b, 4, b.length - 4), result);
    }

    function slice(
        bytes memory _bytes,
        uint256 _start,
        uint256 _length
    ) internal pure returns (bytes memory) {
        require(_length + 31 >= _length, "slice_overflow");
        require(_bytes.length >= _start + _length, "slice_outOfBounds");

        bytes memory tempBytes;

        assembly {
            switch iszero(_length)
            case 0 {
                tempBytes := mload(0x40)
                let lengthmod := and(_length, 31)
                let mc := add(
                    add(tempBytes, lengthmod),
                    mul(0x20, iszero(lengthmod))
                )
                let end := add(mc, _length)

                for {
                    let cc := add(
                        add(
                            add(_bytes, lengthmod),
                            mul(0x20, iszero(lengthmod))
                        ),
                        _start
                    )
                } lt(mc, end) {
                    mc := add(mc, 0x20)
                    cc := add(cc, 0x20)
                } {
                    mstore(mc, mload(cc))
                }

                mstore(tempBytes, _length)
                mstore(0x40, and(add(mc, 31), not(31)))
            }
            default {
                tempBytes := mload(0x40)
                mstore(tempBytes, 0)

                mstore(0x40, add(tempBytes, 0x20))
            }
        }

        return tempBytes;
    }
}
