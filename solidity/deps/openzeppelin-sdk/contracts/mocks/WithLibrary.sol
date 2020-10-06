pragma solidity ^0.5.0;

library MyLibrary {
  function answer() external pure returns(uint256) {
    return 42;
  }
}

contract WithLibraryMock {
  function reply() external pure returns(uint256) {
    return MyLibrary.answer();
  }
}