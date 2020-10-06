pragma solidity ^0.6.0;

import "../deps/openzeppelin-sdk/packages/lib/contracts/upgradeability/AdminUpgradeabilityProxy.sol";

contract Registry is AdminUpgradeabilityProxy {
  constructor(address _logic, address _proxyAdmin, bytes memory _data)
    AdminUpgradeabilityProxy(_logic, _proxyAdmin, _data)
    public
    payable
  {

  }
}