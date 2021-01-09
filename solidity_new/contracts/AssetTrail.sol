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

    event DescribedStructuredAssetDefinitionCreated (
        bytes32 assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bool isContentUnique,
        bytes32 descriptionSchemaHash,
        bytes32 contentSchemaHash,
        uint timestamp
    );
    
    event DescribedUnstructuredAssetDefinitionCreated (
        bytes32 assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bool isContentUnique,
        bytes32 descriptionSchemaHash,
        uint timestamp
    );
    
    event StructuredAssetDefinitionCreated (
        bytes32 assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bool isContentUnique,
        bytes32 contentSchemaHash,
        uint timestamp
    );

    event UnstructuredAssetDefinitionCreated (
        bytes32 assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bool isContentUnique,
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
        address recipient,
        uint amount,
        bytes32 descriptionHash,
        uint timestamp
    );
    
    event PaymentInstanceCreated (
        bytes32 paymentInstanceID,
        bytes32 paymentDefinitionID,
        address author,
        address recipient,
        uint amount,
        uint timestamp
    );

    event AssetInstancePropertySet (
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
    
    function createDescribedStructuredAssetDefinition(bytes32 assetDefinitionID, string memory name, bool isContentPrivate, bool isContentUnique, bytes32 descriptionSchemaHash, bytes32 contentSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        emit DescribedStructuredAssetDefinitionCreated(assetDefinitionID, msg.sender, name, isContentPrivate, isContentUnique, descriptionSchemaHash, contentSchemaHash, now);
    }

    function createDescribedUnstructuredAssetDefinition(bytes32 assetDefinitionID, string memory name, bool isContentPrivate, bool isContentUnique, bytes32 descriptionSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        emit DescribedUnstructuredAssetDefinitionCreated(assetDefinitionID, msg.sender, name, isContentPrivate, isContentUnique, descriptionSchemaHash, now);
    }
    
    function createStructuredAssetDefinition(bytes32 assetDefinitionID, string memory name, bool isContentPrivate, bool isContentUnique, bytes32 contentSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        emit StructuredAssetDefinitionCreated(assetDefinitionID, msg.sender, name, isContentPrivate, isContentUnique, contentSchemaHash, now);
    }
    
    function createUnstructuredAssetDefinition(bytes32 assetDefinitionID, string memory name, bool isContentPrivate, bool isContentUnique) public {
        require(bytes(name).length != 0, "Invalid name");
        emit UnstructuredAssetDefinitionCreated(assetDefinitionID, msg.sender, name, isContentPrivate, isContentUnique, now);
    }

    function createDescribedPaymentDefinition(bytes32 paymentDefinitionID, string memory name, bytes32 descriptionSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        emit DescribedPaymentDefinitionCreated(paymentDefinitionID, msg.sender, name, descriptionSchemaHash, now);
    }

    function createPaymentDefinition(bytes32 paymentDefinitionID, string memory name) public {
        require(bytes(name).length != 0, "Invalid name");
        emit PaymentDefinitionCreated(paymentDefinitionID, msg.sender, name, now);
    }
    
    function createAssetInstanceBatch(bytes32 batchHash) public {
        emit AssetInstanceBatchCreated(batchHash, msg.sender, now);
    }

    function createDescribedPaymentInstance(bytes32 paymentInstanceID, bytes32 paymentDefinitionID, address recipient, uint amount, bytes32 descriptionHash) public {
        require(msg.sender != recipient, "Author and recipient cannot be the same");
        require(amount > 0, "Amount must be greater than 0");
        require(payment.transferFrom(msg.sender, recipient, amount), "Failed to transfer tokens");
        emit DescribedPaymentInstanceCreated(paymentInstanceID, paymentDefinitionID, msg.sender, recipient, amount, descriptionHash, now);
    }

    function createPaymentInstance(bytes32 paymentInstanceID, bytes32 paymentDefinitionID, address recipient, uint amount) public {
        require(msg.sender != recipient, "Author and recipient cannot be the same");
        require(amount > 0, "Amount must be greater than 0");
        require(payment.transferFrom(msg.sender, recipient, amount), "Failed to transfer tokens");
        emit PaymentInstanceCreated(paymentInstanceID, paymentDefinitionID, msg.sender, recipient, amount, now);
    }
    
    function setAssetInstanceProperty(bytes32 assetInstanceID, string memory key, string memory value) public {
        require(bytes(key).length > 0, "Invalid key");
        emit AssetInstancePropertySet(assetInstanceID, msg.sender, key, value, now);
    }
    
}
