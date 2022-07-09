// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

contract Firefly {

    event BatchPin (
        address author,
        uint timestamp,
        string action,
        bytes32 uuids,
        bytes32 batchHash,
        string payloadRef,
        bytes32[] contexts
    );

    function pinBatch(bytes32 uuids, bytes32 batchHash, string memory payloadRef, bytes32[] memory contexts) public {
        emit BatchPin(msg.sender, block.timestamp, "", uuids, batchHash, payloadRef, contexts);
    }

    function networkAction(string memory action, string memory payload) public {
        bytes32[] memory contexts;
        emit BatchPin(msg.sender, block.timestamp, action, 0, 0, payload, contexts);
    }

    function networkVersion() public pure returns (uint8) {
        return 2;
    }
}
