// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

contract Firefly {

    event BatchPin (
        address author,
        uint timestamp,
        bytes32 txnId,
        bytes32 batchId,
        bytes32 payloadRef,
        bytes32[] sequenceHashes
    );

    function pinBatch(bytes32 txnId, bytes32 batchId, bytes32 payloadRef, bytes32[] memory sequenceHashes) public {
        emit BatchPin(msg.sender, block.timestamp, txnId, batchId, payloadRef, sequenceHashes);
    }

}
