pragma solidity ^0.6.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract AssetTrail {

    event MemberRegistered (
        address member,
        string name,
        string assetTrailInstanceID,
        string app2appDestination,
        string docExchangeDestination,
        uint timestamp
    );

    event AssetDefinitionCreated (
        bytes32 assetDefinitionHash,
        address author,
        uint timestamp
    );

    event DescribedAssetInstanceCreated (
        bytes32 assetInstanceID,
        bytes32 assetDefinitionID,
        address author,
        bytes32 descriptionHash,
        bytes32 contentHash,
        uint timestamp
    );

    event AssetInstanceCreated (
        bytes32 assetInstanceID,
        bytes32 assetDefinitionID,
        address author,
        bytes32 contentHash,
        uint timestamp
    );

    event AssetInstanceBatchCreated (
        bytes32 batchHash,
        address author,
        uint timestamp
    );

    event DescribedPaymentDefinitionCreated (
        bytes32 paymentDefinitionID,
        address author,
        string name,
        bytes32 descriptionSchemaHash,
        uint timestamp
    );

    event PaymentDefinitionCreated (
        bytes32 paymentDefinitionID,
        address author,
        string name,
        uint timestamp
    );

    event DescribedPaymentInstanceCreated (
        bytes32 paymentInstanceID,
        bytes32 paymentDefinitionID,
        address author,
        address member,
        uint amount,
        bytes32 descriptionHash,
        uint timestamp
    );

    event PaymentInstanceCreated (
        bytes32 paymentInstanceID,
        bytes32 paymentDefinitionID,
        address author,
        address member,
        uint amount,
        uint timestamp
    );

    event AssetInstancePropertySet (
        bytes32 assetDefinitionID,
        bytes32 assetInstanceID,
        address author,
        string key,
        string value,
        uint timestamp
    );

    ERC20 payment;

    constructor(address paymentContract) public {
        payment = ERC20(paymentContract);
    }

    function registerMember(string memory name, string memory assetTrailInstanceID, string memory app2appDestination, string memory docExchangeDestination) public {
        require(bytes(name).length != 0, "Invalid name");
        emit MemberRegistered(msg.sender, name, assetTrailInstanceID, app2appDestination, docExchangeDestination, now);
    }

    function createAssetDefinition(bytes32 assetDefinitionHash) public {
        emit AssetDefinitionCreated(assetDefinitionHash, msg.sender, now);
    }

    function createDescribedPaymentDefinition(bytes32 paymentDefinitionID, string memory name, bytes32 descriptionSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        emit DescribedPaymentDefinitionCreated(paymentDefinitionID, msg.sender, name, descriptionSchemaHash, now);
    }

    function createPaymentDefinition(bytes32 paymentDefinitionID, string memory name) public {
        require(bytes(name).length != 0, "Invalid name");
        emit PaymentDefinitionCreated(paymentDefinitionID, msg.sender, name, now);
    }

    function createDescribedAssetInstance(bytes32 assetInstanceID, bytes32 assetDefinitionID, bytes32 descriptionHash, bytes32 contentHash) public {
        emit DescribedAssetInstanceCreated(assetInstanceID, assetDefinitionID, msg.sender, descriptionHash, contentHash, now);
    }

    function createAssetInstance(bytes32 assetInstanceID, bytes32 assetDefinitionID, bytes32 contentHash) public {
        emit AssetInstanceCreated(assetInstanceID, assetDefinitionID, msg.sender, contentHash, now);
    }

    function createAssetInstanceBatch(bytes32 batchHash) public {
        emit AssetInstanceBatchCreated(batchHash, msg.sender, now);
    }

    function createDescribedPaymentInstance(bytes32 paymentInstanceID, bytes32 paymentDefinitionID, address member, uint amount, bytes32 descriptionHash) public {
        require(msg.sender != member, "Author and member cannot be the same");
        require(amount > 0, "Amount must be greater than 0");
        require(payment.transferFrom(msg.sender, member, amount), "Failed to transfer tokens");
        emit DescribedPaymentInstanceCreated(paymentInstanceID, paymentDefinitionID, msg.sender, member, amount, descriptionHash, now);
    }

    function createPaymentInstance(bytes32 paymentInstanceID, bytes32 paymentDefinitionID, address member, uint amount) public {
        require(msg.sender != member, "Author and member cannot be the same");
        require(amount > 0, "Amount must be greater than 0");
        require(payment.transferFrom(msg.sender, member, amount), "Failed to transfer tokens");
        emit PaymentInstanceCreated(paymentInstanceID, paymentDefinitionID, msg.sender, member, amount, now);
    }

    function setAssetInstanceProperty(bytes32 assetDefinitionID, bytes32 assetInstanceID, string memory key, string memory value) public {
        require(bytes(key).length > 0, "Invalid key");
        emit AssetInstancePropertySet(assetDefinitionID, assetInstanceID, msg.sender, key, value, now);
    }

}
