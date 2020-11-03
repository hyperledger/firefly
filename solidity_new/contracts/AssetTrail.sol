pragma solidity ^0.6.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract AssetTrail {

    event MemberRegistered (
        address member,
        string name,
        string app2appDestination,
        string docExchangeDestination,
        uint timestamp
    );

    event DescribedStructuredAssetDefinitionCreated (
        uint assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bytes32 descriptionSchemaHash,
        bytes32 contentSchemaHash,
        uint timestamp
    );
    
    event DescribedUnstructuredAssetDefinitionCreated (
        uint assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bytes32 descriptionSchemaHash,
        uint timestamp
    );
    
    event StructuredAssetDefinitionCreated (
        uint assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        bytes32 contentSchemaHash,
        uint timestamp
    );

    event UnstructuredAssetDefinitionCreated (
        uint assetDefinitionID,
        address author,
        string name,
        bool isContentPrivate,
        uint timestamp
    );

    event DescribedAssetInstanceCreated (
        uint assetDefinitionID,
        uint assetInstanceID,
        address author,
        bytes32 descriptionHash,
        bytes32 contentHash,
        uint timestamp
    );

    event AssetInstanceCreated (
        uint assetDefinitionID,
        uint assetInstanceID,
        address author,
        bytes32 contentHash,
        uint timestamp
    );
    
    event DescribedPaymentDefinitionCreated (
        uint paymentDefinitionID,
        address author,
        string name,
        bytes32 descriptionSchemaHash,
        uint amount,
        uint timestamp
    );

    event PaymentDefinitionCreated (
        uint paymentDefinitionID,
        address author,
        string name,
        uint amount,
        uint timestamp
    );

    event PaymentInstanceCreated (
        uint paymentInstanceID,
        uint paymentDefinitionID,
        address author,
        address recipient,
        uint timestamp
    );

    event DescribedPaymentInstanceCreated (
        uint paymentInstanceID,
        uint paymentDefinitionID,
        address author,
        address recipient,
        bytes32 descriptionHash,
        uint timestamp
    );
    
    event AssetPropertySet (
        uint assetInstanceID,
        address propertyAuthor,
        string key,
        string value,
        uint timestamp
    );
    
    uint private assetDefinitionCount;
    uint private paymentDefinitionCount;
    
    uint private assetInstanceCount;
    uint private paymentInstanceCount;

    mapping(string => bool) private assetDefinitions;
    mapping(string => bool) private paymentDefinitions;
    mapping(uint => uint) private paymentAmounts;
    ERC20 payment;

    constructor(address paymentContract) public {
        payment = ERC20(paymentContract);
    }

    function getStatus() public view returns (uint totalAssetDefinitions, uint totalPaymentDefinitions, uint totalAssetInstances, uint totalPaymentInstances) {
        return (assetDefinitionCount, paymentDefinitionCount, assetInstanceCount, paymentInstanceCount);
    }
    
    function registerMember(string memory name, string memory app2appDestination, string memory docExchangeDestination) public {
        require(bytes(name).length != 0, "Invalid name");
        emit MemberRegistered(msg.sender, name, app2appDestination, docExchangeDestination, now);
    }
    
    function createDescribedStructuredAssetDefinition(string memory name, bool isContentPrivate, bytes32 descriptionSchemaHash, bytes32 contentSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit DescribedStructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, descriptionSchemaHash, contentSchemaHash, now);
    }

    function createDescribedUnstructuredAssetDefinition(string memory name, bool isContentPrivate, bytes32 descriptionSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit DescribedUnstructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, descriptionSchemaHash, now);
    }
    
    function createStructuredAssetDefinition(string memory name, bool isContentPrivate, bytes32 contentSchemaHash) public {
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit StructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, contentSchemaHash, now);
    }
    
    function createUnstructuredAssetDefinition(string memory name, bool isContentPrivate) public {
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit UnstructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, now);
    }

    function createDescribedPaymentDefinition(string memory name, bytes32 descriptionSchemaHash, uint amount) public {
        require(bytes(name).length != 0, "Invalid name");
        require(amount > 0 , "Invalid amount");
        require(paymentDefinitions[name] == false, "Payment definition name conflict");
        paymentAmounts[paymentDefinitionCount] = amount;
        paymentDefinitions[name] = true;
        emit DescribedPaymentDefinitionCreated(paymentDefinitionCount++, msg.sender, name, descriptionSchemaHash, amount, now);
    }

    function createPaymentDefinition(string memory name, uint amount) public {
        require(bytes(name).length != 0, "Invalid name");
        require(amount > 0 , "Invalid amount");
        require(paymentDefinitions[name] == false, "Payment definition name conflict");
        paymentAmounts[paymentDefinitionCount] = amount;
        paymentDefinitions[name] = true;
        emit PaymentDefinitionCreated(paymentDefinitionCount++, msg.sender, name, amount, now);
    }
    
    function createDescribedAssetInstance(uint assetDefinitionID, bytes32 descriptionHash, bytes32 contentHash) public {
        emit DescribedAssetInstanceCreated(assetDefinitionID, assetInstanceCount++, msg.sender, descriptionHash, contentHash, now);
    }
    
    function createAssetInstance(uint assetDefinitionID, bytes32 contentHash) public {
        emit AssetInstanceCreated(assetDefinitionID, assetInstanceCount++, msg.sender, contentHash, now);
    }

    function createDescribedPaymentInstance(uint paymentDefinitionID, address recipient, bytes32 descriptionHash) public {
        require(msg.sender != recipient, "Author and recipient cannot be the same");
        require(payment.transferFrom(msg.sender, recipient, paymentAmounts[paymentDefinitionID]), "Failed to transfer tokens");
        emit DescribedPaymentInstanceCreated(paymentInstanceCount++, paymentDefinitionID, msg.sender, recipient, descriptionHash, now);
    }

    function createPaymentInstance(uint paymentDefinitionID, address recipient) public {
        require(msg.sender != recipient, "Author and recipient cannot be the same");
        require(payment.transferFrom(msg.sender, recipient, paymentAmounts[paymentDefinitionID]), "Failed to transfer tokens");
        emit PaymentInstanceCreated(paymentInstanceCount++, paymentDefinitionID, msg.sender, recipient, now);
    }
    
    function setAssetProperty(uint assetInstanceID, string memory key, string memory value) public {
        require(bytes(key).length > 0, "Invalid key");
        emit AssetPropertySet(assetInstanceID, msg.sender, key, value, now);
    }
    
}
