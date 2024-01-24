// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

contract Reverter {
    function goBang() pure public {
        revert("Bang!");
    }
}