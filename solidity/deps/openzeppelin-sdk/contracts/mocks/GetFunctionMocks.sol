pragma solidity ^0.5.0;

contract GetFunctionBase {
  uint256 x;
  
  function initialize(uint256 _x) public {
    x = _x;
  }

  function initialize(string memory _x) public {
    x = bytes(_x).length;
  }

  function initialize(uint256 _x, uint256 _y) public {
    x = _x + _y;
  }

  function another(uint256 _x) public {
    x = _x;
  }
}

contract GetFunctionChild is GetFunctionBase {
  function initialize(bytes memory _x) public {
    x = _x.length;
  }
}

contract GetFunctionOtherChild is GetFunctionBase {
  function initialize(bytes32 _x) public {
    x = uint256(_x);
  }
}

contract GetFunctionGrandchild is GetFunctionChild, GetFunctionOtherChild { }

contract GetFunctionOtherGrandchild is GetFunctionOtherChild, GetFunctionChild { }