// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

contract Firefly {

    event BatchPin (
        address author,
        uint timestamp,
        string namespace,
        bytes32 uuids,
        bytes32 batchHash,
        string payloadRef,
        bytes32[] contexts
    );

    function pinBatch(string memory namespace, bytes32 uuids, bytes32 batchHash, string memory payloadRef, bytes32[] memory contexts) public {
        emit BatchPin(msg.sender, block.timestamp, namespace, uuids, batchHash, payloadRef, contexts);
    }

}
