// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

import "./IBatchPin.sol";

contract Firefly is IBatchPin {
    function pinBatchData(bytes calldata data) public override {
        bytes32 uuids;
        bytes32 batchHash;
        string memory payloadRef;
        bytes32[] memory contexts;
        (uuids, batchHash, payloadRef, contexts) = abi.decode(
            data,
            (bytes32, bytes32, string, bytes32[])
        );
        emit BatchPin(
            tx.origin,
            block.timestamp,
            "firefly:contract_invoke_pin",
            uuids,
            batchHash,
            payloadRef,
            contexts
        );
    }

    function pinBatch(
        bytes32 uuids,
        bytes32 batchHash,
        string memory payloadRef,
        bytes32[] memory contexts
    ) public override {
        emit BatchPin(
            tx.origin,
            block.timestamp,
            "firefly:batch_pin",
            uuids,
            batchHash,
            payloadRef,
            contexts
        );
    }

    function networkAction(string memory action, string memory payload) public {
        bytes32[] memory contexts;
        emit BatchPin(
            tx.origin,
            block.timestamp,
            action,
            0,
            0,
            payload,
            contexts
        );
    }

    function networkVersion() public pure returns (uint8) {
        return 2;
    }
}
