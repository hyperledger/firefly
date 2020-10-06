pragma experimental ABIEncoderV2;

contract WithStructInConstructor {

  struct Data {
    string foo;
    string bar;
    uint256 buz;
  }

  Data public localData;
  address public sender;

  constructor(Data memory _data) public {
    localData.foo = _data.foo;
    localData.bar = _data.bar;
    localData.buz = _data.buz;
    sender = msg.sender;
  }

  function foo() public view returns (string memory) {
    return localData.foo;
  }

  function bar() public view returns (string memory) {
    return localData.bar;
  }

  function buz() public view returns (uint256) {
    return localData.buz;
  }
}
