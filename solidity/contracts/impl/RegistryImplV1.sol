pragma solidity ^0.6.0;

import "@openzeppelin/contracts-ethereum-package/contracts/Initializable.sol";

contract RegistryImplV1 is Initializable {
    
  struct Member {
    address member;
    string name;
    string destination;
    uint timestamp;
  }
  
  struct Entry {
    address author;
    bytes32 contentHash;
    uint timestamp;
    mapping(address => uint) subscriptions;
    mapping(bytes32 => Attachment) attachments;
    mapping(address => mapping(string => string)) metadata;
  }

  struct Attachment {
      address author;
      string description;
      uint timestamp;
  }

  event MemberRegistered (
      address member,
      string name,
      string destination,
      uint timestamp
  );

  event EntryCreated (
      bytes32 keyHash,
      address author,
      bytes32 contentHash,
      uint timestamp
  );

  event SubscriptionAdded (
      bytes32 keyHash,
      address subscriber,
      uint timestamp
  );
  
  event AttachmentAdded (
      bytes32 keyHash,
      bytes32 attachmentHash,
      address author,
      string description,
      uint timestamp
  );
      
  event MetadataAdded (
      bytes32 keyHash,
      address author,
      string key,
      string value,
      uint timestamp
  );

  mapping(address => Member) private members;
  mapping(bytes32 => Entry) private entries;

  function initialize() public initializer {
  }
  
  function registerMember(string memory name, string memory destination) public {
      require(bytes(name).length != 0, "Invalid member name");
      require(bytes(destination).length != 0, "Invalid member destination");
      require(members[msg.sender].timestamp == 0, "Member already registered");
      uint timestamp = now;
      members[msg.sender] = Member(msg.sender, name, destination, timestamp);
      emit MemberRegistered(msg.sender, name, destination, timestamp);
  }
  
  function createEntry(bytes32 keyHash, bytes32 contentHash) public {
      require(bytes(members[msg.sender].name).length > 0, "Member must be registered");
      require(entries[keyHash].timestamp == 0, "Entry already exists");
      uint timestamp = now;
      entries[keyHash] = Entry(msg.sender, contentHash, timestamp);
      emit EntryCreated(keyHash, msg.sender, contentHash, timestamp);
  }
  
  function addEntrySubscriber(bytes32 keyHash, address subscriber) public {
      require(bytes(members[msg.sender].name).length > 0, "Member must be registered");
      require(bytes(members[subscriber].name).length > 0, "Subscriber must be registered");
      require(entries[keyHash].timestamp != 0, "Entry not found");
      require(entries[keyHash].author == msg.sender, "Must be entry author to add subsribers");
      require(entries[keyHash].author != subscriber, "Cannot subscribe to authored entry");
      require(entries[keyHash].subscriptions[subscriber] == 0, "Already subscribed");
      uint timestamp = now;
      entries[keyHash].subscriptions[subscriber] = timestamp;
      emit SubscriptionAdded(keyHash, subscriber, timestamp);
  }
  
  function addAttachment(bytes32 keyHash, bytes32 attachmentHash, string memory description) public {
      require(bytes(description).length > 0, "Description cannot be empty");
      require(entries[keyHash].timestamp != 0, "Entry not found");
      require(entries[keyHash].attachments[attachmentHash].timestamp == 0, "Attachment already exists");
      require(entries[keyHash].author == msg.sender || entries[keyHash].subscriptions[msg.sender] != 0, "Must be entry author or subscriber to add attachment");
      uint timestamp = now;
      entries[keyHash].attachments[attachmentHash] = Attachment(msg.sender, description, timestamp);
      emit AttachmentAdded(keyHash, attachmentHash, msg.sender, description, timestamp);
  }
  
  function addMetadata(bytes32 keyHash, string memory key, string memory value) public {
      require(bytes(key).length > 0, "Key cannot be empty");
      require(bytes(value).length > 0, "Value cannot be empty");
      require(entries[keyHash].timestamp != 0, "Entry not found");
      require(entries[keyHash].author == msg.sender || entries[keyHash].subscriptions[msg.sender] != 0, "Must be entry author or subscriber to add metadata");
      require(keccak256(bytes(entries[keyHash].metadata[msg.sender][key])) != keccak256(bytes(value)), "Metadata already set");
      entries[keyHash].metadata[msg.sender][key] = value;
      emit MetadataAdded(keyHash, msg.sender, key, value, now);
  }

}