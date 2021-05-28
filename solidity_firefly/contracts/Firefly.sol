// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

contract Firefly {

    event BroadcastBatch (
        address author,
        uint timestamp,
        bytes32 txnId,
        bytes32 batchId,
        bytes32 payloadRef
    );

    function broadcastBatch(bytes32 txnId, bytes32 batchId, bytes32 payloadRef) public {
        emit BroadcastBatch(msg.sender, block.timestamp, txnId, batchId, payloadRef);
    }

}
