pragma solidity ^0.5.0;

import "../Initializable.sol";

contract WithInitialize {
  uint public value;

  function initialize() public {
    value = 42;
  }
}

contract AnotherWithInitialize is Initializable {
  uint public anotherValue;

  function init() initializer public {
    anotherValue = 42;
  }
}

contract WithoutInitialize {
  function say() public pure returns (string memory) {
    return "WithoutInitialize";
  }
}

contract WithBaseUninitialized is WithInitialize, AnotherWithInitialize {
  uint public someValue;

  function initialize() public {
    someValue = 42;
  }
}

contract WithBaseInitialized is WithInitialize, AnotherWithInitialize {
  uint public someValue;

  function initialize() public {
    WithInitialize.initialize();
    AnotherWithInitialize.init();
    someValue = 42;
  }
}

contract WithSimpleBaseUninitialized is WithoutInitialize {
  uint public someValue;

  function initialize() public {
    someValue = 42;
  }
}

contract ShouldHaveInitialize is WithInitialize, AnotherWithInitialize {
  uint public someValue;
}

contract DoesNotNeedAnInitialize is WithInitialize, WithoutInitialize {
  uint public someValue;
}

contract ExtendsFromShouldHaveInitialize is ShouldHaveInitialize {
  uint public someOtherValue;
}
