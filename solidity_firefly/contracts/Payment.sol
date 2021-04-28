pragma solidity ^0.6.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract Payment is ERC20 {
        constructor(uint256 initialSupply) public ERC20("Asset Trail Token", "ATT") {
        _mint(msg.sender, initialSupply);
    }
}
