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
        address author,
        bytes32 descriptionHash,
        bytes32 contentHash,
        uint timestamp
    );

    event AssetInstanceCreated (
        uint assetDefinitionID,
        address author,
        bytes32 contentHash,
        uint timestamp
    );
    
    event PaymentDefinitionCreated (
        uint paymentDefinitionID,
        address author,
        string name,
        bytes32 paymentSchema,
        uint amount,
        uint timestamp
    );
    
    event PaymentInstanceCreated (
        uint paymentDefinitionID,
        address author,
        address recipient,
        bytes32 paymentHash,
        uint timestamp
    );
    
    event AssetPropertySet (
        address propertyAuthor,
        uint assetDefinitionID,
        bytes32 assetContentHash,
        address assetAuthor,
        uint assetTimestamp,
        string key,
        string value,
        uint timestamp
    );
    
    uint private assetDefinitionCount;
    uint private paymentDefinitionCount;
    
    uint private assetInstanceCount;
    uint private paymentInstanceCount;

    mapping(address => bool) private members;
    mapping(string => bool) private assetDefinitions;
    mapping(string => bool) private paymentDefinitions;
    mapping(uint => uint) private paymentAmounts;
    ERC20 payment;

    constructor(address paymentContract) public {
        payment = ERC20(paymentContract);
    }
    
    function registerMember(string memory name, string memory app2appDestination, string memory docExchangeDestination) public {
        require(bytes(name).length != 0, "Invalid name");
        members[msg.sender] = true;
        emit MemberRegistered(msg.sender, name, app2appDestination, docExchangeDestination, now);
    }
    
    function createDescribedStructuredAssetDefinition(string memory name, bool isContentPrivate, bytes32 descriptionSchemaHash, bytes32 contentSchemaHash) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit DescribedStructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, descriptionSchemaHash, contentSchemaHash, now);
    }

    function createDescribedUnstructuredAssetDefinition(string memory name, bool isContentPrivate, bytes32 descriptionSchemaHash) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit DescribedUnstructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, descriptionSchemaHash, now);
    }
    
    function createStructuredAssetDefinition(string memory name, bool isContentPrivate, bytes32 contentSchemaHash) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit StructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, contentSchemaHash, now);
    }
    
    function createUnstructuredAssetDefinition(string memory name, bool isContentPrivate) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(bytes(name).length != 0, "Invalid name");
        require(assetDefinitions[name] == false, "Asset definition name conflict");
        assetDefinitions[name] = true;
        emit UnstructuredAssetDefinitionCreated(assetDefinitionCount++, msg.sender, name, isContentPrivate, now);
    }

    function createPaymentDefinition(string memory name, bytes32 paymentSchema, uint amount) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(bytes(name).length != 0, "Invalid name");
        require(amount > 0 , "Invalid amount");
        require(paymentDefinitions[name] == false, "Payment definition name conflict");
        paymentAmounts[paymentDefinitionCount] = amount;
        paymentDefinitions[name] = true;
        emit PaymentDefinitionCreated(paymentDefinitionCount++, msg.sender, name, paymentSchema, amount, now);
    }
    
    function createDescribedAssetInstance(uint assetDefinitionID, bytes32 descriptionHash, bytes32 contentHash) public {
        require(members[msg.sender] == true, "Member must be registered");
        assetInstanceCount++;
        emit DescribedAssetInstanceCreated(assetDefinitionID, msg.sender, descriptionHash, contentHash, now);
    }
    
    function createAssetInstance(uint assetDefinitionID, bytes32 contentHash) public {
        require(members[msg.sender] == true, "Member must be registered");
        assetInstanceCount++;
        emit AssetInstanceCreated(assetDefinitionID, msg.sender, contentHash, now);
    }

    function createPaymentInstance(uint paymentDefinitionID, address recipient, bytes32 content) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(members[recipient] == true, "Recipient must be registered");
        require(msg.sender != recipient, "Author and recipient cannot be the same");
        require(payment.transferFrom(msg.sender, recipient, paymentAmounts[paymentDefinitionID]), "Failed to transfer tokens");
        paymentInstanceCount++;
        emit PaymentInstanceCreated(paymentDefinitionID, msg.sender, recipient, content, now);
    }
    
    function setAssetProperty(uint assetDefinitionID, bytes32 assetContentHash, address assetAuthor, uint assetTimestamp, string memory key, string memory value) public {
        require(members[msg.sender] == true, "Member must be registered");
        require(members[assetAuthor] == true, "Asset author must be registered");
        require(bytes(key).length > 0, "Invalid key");
        emit AssetPropertySet(msg.sender, assetDefinitionID, assetContentHash, assetAuthor, assetTimestamp, key, value, now);
    }
    
}
