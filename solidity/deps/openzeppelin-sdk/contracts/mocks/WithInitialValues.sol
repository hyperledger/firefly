pragma solidity ^0.5.0;

contract WithoutInitialValuesInFieldsDeclarations {
  uint constant myConst = 115994;

  function doSomething() public pure returns (string memory) {
    return 'do';
  }
}

contract WithInitialValuesInFieldsDeclarations {
  string public allaps = 'allaps';

  function doSomething() public pure returns (string memory) {
    return 'do';
  }
}

contract WithParentWithInitialValuesInFieldsDeclarations is WithInitialValuesInFieldsDeclarations {
  string public myVar;

  function doSomething() public pure returns (string memory) {
    return 'do';
  }
}

