pragma solidity ^0.6.0;

import "../deps/openzeppelin-sdk/contracts/upgradeability/AdminUpgradeabilityProxy.sol";

contract AssetTrail is AdminUpgradeabilityProxy {
  constructor(address _logic, address _proxyAdmin, bytes memory _data)
    AdminUpgradeabilityProxy(_logic, _proxyAdmin, _data)
    public
    payable
  {

  }
}