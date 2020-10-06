pragma solidity ^0.5.0;

import "./StorageMocks.sol";
import "mock-dependency/contracts/DependencyStorageMock.sol";

contract StorageMockWithReferences {
  StorageMockWithEnums.MyEnum internal my_enum;
  StorageMockWithStructs.MyStruct internal my_struct;
  SimpleStorageMock internal my_contract;
}

contract StorageMockWithNodeModulesReferences {
  DependencyStorageMock.MyEnum internal my_enum;
  DependencyStorageMock.MyStruct internal my_struct;
  DependencyStorageMock internal my_contract;
}

contract StorageMockChainBase {
  uint256 internal base;
}

contract StorageMockChainA1 is StorageMockChainBase {
  uint256 internal a1;
  uint256 internal a2;
}

contract StorageMockChainA2 is StorageMockChainA1 {
  uint256 public a3;
  uint256 public a4;
}

contract StorageMockChainB is StorageMockChainBase {
  uint256 internal b1;
  uint256 internal b2;
}

contract StorageMockChainChild is StorageMockChainA2, StorageMockChainB {
  uint256 internal child;

  function slots() public pure returns(uint256 baseSlot, uint256 a1Slot, uint256 a3Slot, uint256 b1Slot, uint256 childSlot) {
    assembly {
      baseSlot := base_slot
      a1Slot := a1_slot
      a3Slot := a3_slot
      b1Slot := b1_slot
      childSlot := child_slot
    }
  }
}