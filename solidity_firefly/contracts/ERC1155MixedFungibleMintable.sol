// SPDX-License-Identifier: Apache-2.0

pragma solidity ^0.6.0;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "@openzeppelin/contracts/utils/Context.sol";

/**
    @dev Mintable form of ERC1155 with mixed fungible/non-fungible item support.
    Based on reference implementation:
    https://github.com/enjin/erc-1155/blob/master/contracts/ERC1155MixedFungibleMintable.sol
*/
contract ERC1155MixedFungibleMintable is Context, ERC1155 {
    // Use a split bit implementation:
    // Store the type in the upper 128 bits, non-fungible index in the lower 128.
    // The top bit is a flag to tell if this is non-fungible.
    uint256 constant TYPE_MASK = uint256(uint128(~0)) << 128;
    uint256 constant NF_INDEX_MASK = uint128(~0);
    uint256 constant TYPE_NF_BIT = 1 << 255;

    uint256 nonce;
    mapping (uint256 => address) public creators;
    mapping (uint256 => uint256) public maxIndex;

    function isFungible(uint256 id) internal pure returns(bool) {
        return id & TYPE_NF_BIT == 0;
    }
    function isNonFungible(uint256 id) internal pure returns(bool) {
        return id & TYPE_NF_BIT == TYPE_NF_BIT;
    }

    // Only the creator of a token type is allowed to mint it.
    modifier creatorOnly(uint256 type_id) {
        require(creators[type_id] == _msgSender());
        _;
    }

    constructor(string memory uri) ERC1155(uri) public {
    }

    function create(string calldata uri, bool is_fungible)
        external
        virtual
        returns(uint256 type_id)
    {
        type_id = (++nonce << 128);
        if (!is_fungible)
          type_id = type_id | TYPE_NF_BIT;

        creators[type_id] = _msgSender();

        emit TransferSingle(_msgSender(), address(0x0), address(0x0), type_id, 0);

        if (bytes(uri).length > 0)
            emit URI(uri, type_id);
    }

    function mintNonFungible(uint256 type_id, address[] calldata to, bytes calldata data)
        external
        virtual
        creatorOnly(type_id)
    {
        require(isNonFungible(type_id), "ERC1155MixedFungibleMintable: id does not represent a non-fungible type");

        // Indexes are 1-based.
        uint256 index = maxIndex[type_id] + 1;
        maxIndex[type_id] = to.length.add(maxIndex[type_id]);

        for (uint256 i = 0; i < to.length; ++i) {
            address to_address = to[i];
            uint256 id = type_id | index + i;
            _mint(to_address, id, 1, data);
        }
    }

    function mintFungible(uint256 type_id, address[] calldata to, uint256[] calldata amounts, bytes calldata data)
        external
        virtual
        creatorOnly(type_id)
    {
        require(isFungible(type_id), "ERC1155MixedFungibleMintable: id does not represent a fungible type");
        require(to.length == amounts.length, "ERC1155MixedFungibleMintable: to and amounts length mismatch");

        for (uint256 i = 0; i < to.length; ++i) {
            address to_address = to[i];
            uint256 amount = amounts[i];
            _mint(to_address, type_id, amount, data);
        }
    }
}
