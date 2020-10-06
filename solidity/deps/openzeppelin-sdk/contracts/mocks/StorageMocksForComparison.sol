pragma solidity ^0.5.0;

contract StorageMockEmpty {
}

contract StorageMockSimpleOriginal {
  uint256 a;
  uint256 b;
}

contract StorageMockSimpleUnchanged {
  uint256 a;
  uint256 b;
}

contract StorageMockSimpleWithAddedVar {
  uint256 a;
  uint256 b;
  uint256 c;
}

contract StorageMockSimpleWithInsertedVar {
  uint256 a;
  uint256 c;
  uint256 b;
}

contract StorageMockSimpleWithUnshiftedVar {
  uint256 c;
  uint256 a;
  uint256 b;
}

contract StorageMockSimpleWithRenamedVar {
  uint256 a;
  uint256 c;
}

contract StorageMockSimpleWithTypeChanged {
  uint256 a;
  string b;
}

contract StorageMockSimpleWithDeletedVar {
  uint256 b;
}

contract StorageMockSimpleWithPoppedVar {
  uint256 a;
}

contract StorageMockSimpleWithReplacedVar {
  uint256 a;
  string c;
}

contract StorageMockSimpleChangedWithAppendedVar {
  uint256 a2;
  uint256 b2;
  uint256 c2;
}

contract StorageMockComplexOriginal {
  mapping(address => uint256) a;
}

contract StorageMockComplexWithChangedVar {
  mapping(address => address) a;
}

contract StorageMockChainPrivateBase {
  uint256 private a;
}

contract StorageMockChainPrivateChildV1 is StorageMockChainPrivateBase  {
}

contract StorageMockChainPrivateChildV2 is StorageMockChainPrivateBase {
  uint256 private a;
}
