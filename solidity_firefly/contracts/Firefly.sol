// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

contract Firefly {

    event BatchPin (
        address author,
        uint timestamp,
        bytes32 uuids,
        bytes32 batchHash,
        bytes32 payloadRef,
        bytes32[] pins
    );

    function pinBatch(bytes32 uuids, bytes32 batchHash, bytes32 payloadRef, bytes32[] memory pins) public {
        emit BatchPin(msg.sender, block.timestamp, uuids, batchHash, payloadRef, pins);
    }

}
