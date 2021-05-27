// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.6.0 <0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract Firefly {

    event BroadcastBatch (
        address author,
        uint timestamp,
        bytes32 txnId,
        bytes32 batchId,
        bytes32 payloadRef
    );

    ERC20 payment;

    constructor(address paymentContract) public {
        payment = ERC20(paymentContract);
    }

    function broadcastBatch(bytes32 txnId, bytes32 batchId, bytes32 payloadRef) public {
        emit BroadcastBatch(msg.sender, block.timestamp, txnId, batchId, payloadRef);
    }

}
