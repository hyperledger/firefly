A contract listener configures FireFly to stream events from the blockchain,
from a specific location on the blockchain, according to a given definition
of the interface for that event.

Check out the [Custom Contracts Tutorial](../../tutorials/custom_contracts/index.md) for
a walk-through of how to set up listeners for the events from your smart contracts.

See below for a deep dive into the format of contract listeners and important concepts to understand when managing them.

### Event filters

#### Multiple filters

From v1.3.1 onwards, a contract listener can be created with multiple filters under a single topic, when supported by the connector. Each filter contains:

- a reference to a specific blockchain event to listen for
- (optional) a specific location/address to listen from
- a connector-specific signature (generated from the event and the location)

In addition to this list of multiple filters, the listener specifies a single `topic` to identify the stream of events.

Creating a single listener that listens for multiple events will allow for the easiest management of listeners, and for strong ordering of the events that they process.

#### Single filter

Before v1.3.1, each contract listener would only support listening to one specific event from a contract interface. Each listener would be comprised of:

- a reference to a specific blockchain event to listen for
- (optional) a specific location/address to listen from
- a connector-specific signature (generated from the event), which allows you to easily identify and search for the contact listener for an event
- a `topic` which determines the ordered stream that these events are part of

For backwards compatibility, this format is still supported by the API.

### Signature strings

#### String format

Each filter is identified by a generated `signature` that matches a single event, and each contract listener is identified by a `signature` computed from its filters.

Ethereum provides a string standard for event signatures, of the form `EventName(uint256,bytes)`. Prior to v1.3.1, the signature of each Ethereum contract listener would exactly follow this Ethereum format.

As of v1.3.1, Ethereum format signature strings have been changed in FireFly, because this format does not fully describe the event - particularly because each top-level parameter can in the ABI definition be marked as `indexed`. For example, while the following two Solidity events have the same signature, they are serialized differently due to the different placement of `indexed` parameters, and thus a listener must define both individually to be able to process them:

- ERC-20 `Transfer`

  ```solidity
  event Transfer(address indexed _from, address indexed _to, uint256 _value)
  ```

- ERC-721 `Transfer`

  ```solidity
  event Transfer(address indexed _from, address indexed _to, uint256 indexed _tokenId);
  ```

The two above are now expressed in the following manner by the FireFly Ethereum blockchain connector:

```solidity
Transfer(address,address,uint256) [i=0,1]
Transfer(address,address,uint256) [i=0,1,2]
```

The `[i=]` listing at the end of the signature indicates the position of all parameters that are marked as `indexed`.

Building on the blockchain-specific signature format for each event, FireFly will then compute the final signature for each filter and each contract listener as follows:

- Each filter signature is a combination of the location and the specific connector event signature, such as `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1:Transfer(address,address,uint256) [i=0,1]`
- Each contract listener signature is a concatenation of all the filter signatures, separated by `;`

#### Duplicate filters

FireFly restricts the creation of a contract listener containing duplicate filters.

This includes the special case where one filter is a superset of another filter, due to a wildcard location.

For example, if two filters are listening to the same event, but one has specified a location and the other hasn't, then the latter will be a superset, and already be listening to all the events matching the first filter. Creation of duplicate or superset filters within a single listener will be blocked.

#### Duplicate listeners

As noted above, each listener has a generated signature. This signature - containing all the locations and event signatures combined with the listener topic - will guarantee uniqueness of the contract listener. If you tried to create the same listener again, you would receive HTTP 409. This combination can allow a developer to assert that their listener exists, without the risk of creating duplicates.

**Note:** Prior to v1.3.1, FireFly would detect duplicates simply by requiring a unique combination of signature + topic + location for each listener. The updated behavior for the listener signature is intended to preserve similar functionality, even when dealing with listeners that contain many event filters.

### Backwards compatibility

As noted throughout this document, the behavior of listeners has changed in v1.3.1. However, the following behaviors are retained for backwards-compatibility, to ensure that code written prior to v1.3.1 should continue to function.

- The response from all query APIs of `listeners` will continue to populate top-level `event` and `location` fields
  - The first entry from the `filters` array is duplicated to these fields
- On input to create a new `listener`, the `event` and `location` fields are still supported
  - They function identically to supplying a `filters` array with a single entry
- The `signature` field is preserved at the listener level
  - The format has been changed as described above

## Input examples

The two input formats supported when creating a contract listener are shown below.

### With event definition

In these examples, the event schema in the FireFly Interface format is provided describing the event and its parameters. See [FireFly Interface Format](../firefly_interface_format.md)

**Muliple Filters**
```json
{
  "filters": [
    {
      "event": {
          "name": "Changed",
          "description": "",
          "params": [
              {
                  "name": "x",
                  "schema": {
                      "type": "integer",
                      "details": {
                          "type": "uint256",
                          "internalType": "uint256"
                      }
                  }
              }
          ]
      },
      "location": {
        "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
      }
    },
    {
      "event": {
          "name": "AnotherEvent",
          "description": "",
          "params": [
              {
                  "name": "my-field",
                  "schema": {
                      "type": "string",
                      "details": {
                        "type": "address",
                        "internalType": "address",
                        "indexed": true
                      }
                  }
              }
          ]
      },
      "location": {
        "address": "0xa4ea5d0b6b2eaf194716f0cc73981939dca27da1"
      }
    }
  ],
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```

**One filter (old format)**

```json
{
  "event": {
      "name": "Changed",
      "description": "",
      "params": [
          {
              "name": "x",
              "schema": {
                  "type": "integer",
                  "details": {
                      "type": "uint256",
                      "internalType": "uint256"
                  }
              }
          }
      ]
  },
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```

### With interface reference

These examples use an `interface` reference when creating the filters, the `eventPath` field is used to reference an event defined within the interface provided. In this case, we do not need to provide the event schema as the section above shows. See an example of creating a [FireFly Interface](../../tutorials/custom_contracts/ethereum.md/#the-firefly-interface-format) for an EVM smart contract. 

**Muliple Filters**
```json
{
  "filters": [
    {
      "interface": {
        "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
      },
      "location": {
        "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
      },
      "eventPath": "Changed"
    },
    {
      "interface": {
        "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
      },
      "location": {
        "address": "0xa4ea5d0b6b2eaf194716f0cc73981939dca27da1"
      },
      "eventPath": "AnotherEvent"
    }
  ],
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```

**One filter (old format)**

```json
{
  "interface": {
    "id": "8bdd27a5-67c1-4960-8d1e-7aa31b9084d3"
  },
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "eventPath": "Changed",
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```
