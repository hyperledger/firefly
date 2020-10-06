pragma solidity ^0.5.0;

import "mock-dependency/contracts/Greeter.sol";
import "mock-dependency/contracts/DependencyInvalid.sol";

contract WithConstructor {
  uint256 public value;

  constructor() public {
    value = 42;
  }

  function say() public pure returns (string memory) {
    return "WithConstructor";
  }
}

contract WithParentConstructor is WithConstructor {
}

contract WithAncestorConstructor is WithParentConstructor {
}

contract WithDependencyParentConstructor is DependencyWithConstructor {
}

contract WithEmptyConstructor {
  constructor() internal { }
}

contract WithModifierInConstructor {
  modifier modifies { _; }
  constructor() modifies internal { }
}

contract WithAncestorEmptyConstructor is WithEmptyConstructor {
}

contract WithFailingConstructor {
  constructor() public {
    assert(false);
  }
}

contract WithSelfDestruct {
  uint256 public value;

  constructor() public {
    if (true)
      selfdestruct(msg.sender);
  }

  function say() public pure returns (string memory) {
    return "WithSelfDestruct";
  }
}

contract WithParentWithSelfDestruct is WithSelfDestruct {
  function say() public pure returns (string memory) {
    return "WithParentWithSelfDestruct";
  }
}

contract WithDelegateCall {
  constructor(address _e) public {
    // bytes4(keccak256("kill()")) == 0x41c0e1b5
    bytes memory data = "\x41\xc0\xe1\xb5";
    (bool success,) = _e.delegatecall(data);
    require(success);
  }
  
  function say() public pure returns (string memory) {
    return "WithDelegateCall";
  }
}

contract WithParentWithDelegateCall is WithDelegateCall {
  constructor(address _e) public WithDelegateCall(_e) {
  }

  function say() public pure returns (string memory) {
    return "WithParentWithDelegateCall";
  }
}

contract WithVanillaBaseContract is Greeter { }