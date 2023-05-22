// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.9.0;

interface IBatchPin {
    event BatchPin(
        address author,
        uint timestamp,
        string action,
        bytes32 uuids,
        bytes32 batchHash,
        string payloadRef,
        bytes32[] contexts
    );

    function pinBatchData(bytes calldata data) external;

    function pinBatch(
        bytes32 uuids,
        bytes32 batchHash,
        string memory payloadRef,
        bytes32[] memory contexts
    ) external;
}
