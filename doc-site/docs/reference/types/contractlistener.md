---
title: ContractListener
---
{% include-markdown "./_includes/contractlistener_description.md" %}

See below a deep dive into the format of contract listeners and important concepts to understand when managing them. 

###  Multiple filters

From v1.3.1 onwards, a contract listener can be created with multiple filters under a single topic when supported by the connector. 

Before this change each contract listener would only support listening to one specific event from an interface previously defined. A connector specific event signature would be created to easily identify and search for the contact listener for that event. Within that listener, you could optionally specify a location to listen from. Alongside declaring what and where to listen from, you need to specify a `topic` as describe in (TODO LINK) for the ordering constraint of your application. See below for an example with multiple filters. This format is still supported by the API.

Now from v1.3.1, we have extended the contract listeners API to enable users to supply multiple filters where each contains the event to listen to and the optional location. Each filter will also contain a new connector specific signature which is constructed from the location and the event signature. FireFly will restrict the creation of a contract listener with duplicate filters or supersets. For example, if two filters are listening to the same event but one has specified a location and the other hasn't then the latter will be a superset and already be listening to all the events matching the first filter. 

FireFly made a conscious decision to restricts the creation of duplicate contract listeners and would use the values mentioned earlier to create a unique combination of signature + topic + location. If you tried to create the same listener, you would get a 409. This combination is super useful to a developer to asset that there listener exists. When using muliple filters we can no longer rely on the signature of one event and the location of one filters to calculate uniqueness of a contract listener. As part of v1.3.1, we have come up with a new format where we create a new signature format per filter of location + event signature and then concatinate all of these signatures to build the contract listener signature. This contract listener signature containing all the locations and event signtures combined with the topic will guarantee uniqueness of the contract listener. 

### Signature enhancements

As mentioned above, we have introduced a new format for signatures of contract listener and filter signature:
- Each filter signature will be a combination of the location and the specific connector event signature
- Each contract listener signature will be a concatination of all the filter signatures

Furthemore, we noticed that specifically for Ethereum the ABI does not include a differenciation between indexed and non-indexed fields so we included a new section to the Ethereum specific event signature to add the index of the indexed fields on the signature, as such a filter listening to the event `Changed(address indexed from, uint256 value)` at address `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1` will result in `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1:Changed(address,uint256) [i=0]` where `[i=0]` specifies that the first field is indexed. If there were more indexed fields, it will be a comma separate list of the index of those indexed fields such as `[i=0,2,3]` specifies that fields at index 0, 2, and 3 are indexed fields.

### Example with multiple filters

These examples are using the Ethereum connector.

Request payload

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
    }
  ],
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```


Response payload

Notice that both the 
```json
{
  "id": "e7c8457f-4ffd-42eb-ac11-4ad8aed30de1",
  "interface": {
    "id": "55fdb62a-fefc-4313-99e4-e3f95fcca5f0"
  },
  "namespace": "default",
  "name": "019104d7-bb0a-c008-76a9-8cb923d91b37",
  "backendId": "019104d7-bb0a-c008-76a9-8cb923d91b37",
  "location": {
    "address": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1"
  },
  "created": "2024-07-30T18:12:12.704964Z",
  "event": {
    "name": "Changed",
    "description": "",
    "params": [
      {
        "name": "from",
        "schema": {
          "type": "string",
          "details": {
            "type": "address",
            "internalType": "address",
            "indexed": true
          }
        }
      },
      {
        "name": "value",
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
  "signature": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1:Changed(address,uint256) [i=0]",
  "topic": "simple-storage",
  "options": {
    "firstEvent": "newest"
  },
  "filters": [
    {
      "event": {
        "name": "Changed",
        "description": "",
        "params": [
          {
            "name": "from",
            "schema": {
              "type": "string",
              "details": {
                "type": "address",
                "internalType": "address",
                "indexed": true
              }
            }
          },
          {
            "name": "value",
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
      "interface": {
        "id": "55fdb62a-fefc-4313-99e4-e3f95fcca5f0"
      },
      "signature": "0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1:Changed(address,uint256) [i=0]"
    }
  ]
}
```


### Example with single event (old format)

Request payload

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

```json
{
    "id": "d61980a9-748c-4c72-baf5-8b485b514d59",
    "interface": {
        "id": "ff1da3c1-f9e7-40c2-8d93-abb8855e8a1d"
    },
    "namespace": "ns1",
    "name": "contract1_events",
    "backendId": "sb-dd8795fc-a004-4554-669d-c0cf1ee2c279",
    "location": {
        "address": "0x596003a91a97757ef1916c8d6c0d42592630d2cf"
    },
    "created": "2022-05-16T01:23:15Z",
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
    "signature": "Changed(uint256)",
    "topic": "app1_topic",
    "options": {
        "firstEvent": "newest"
    }
}
```

### Field Descriptions

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the smart contract listener | [`UUID`](simpletypes.md#uuid) |
| `interface` | Deprecated: Please use 'interface' in the array of 'filters' instead | [`FFIReference`](#ffireference) |
| `namespace` | The namespace of the listener, which defines the namespace of all blockchain events detected by this listener | `string` |
| `name` | A descriptive name for the listener | `string` |
| `backendId` | An ID assigned by the blockchain connector to this listener | `string` |
| `location` | Deprecated: Please use 'location' in the array of 'filters' instead | [`JSONAny`](simpletypes.md#jsonany) |
| `created` | The creation time of the listener | [`FFTime`](simpletypes.md#fftime) |
| `event` | Deprecated: Please use 'event' in the array of 'filters' instead | [`FFISerializedEvent`](#ffiserializedevent) |
| `signature` | A concatenation of all the stringified signature of the event and location, as computed by the blockchain plugin | `string` |
| `topic` | A topic to set on the FireFly event that is emitted each time a blockchain event is detected from the blockchain. Setting this topic on a number of listeners allows applications to easily subscribe to all events they need | `string` |
| `options` | Options that control how the listener subscribes to events from the underlying blockchain | [`ContractListenerOptions`](#contractlisteneroptions) |
| `filters` | A list of filters for the contract listener. Each filter is made up of an Event and an optional Location. Events matching these filters will always be emitted in the order determined by the blockchain. | [`ListenerFilter[]`](#listenerfilter) |

## FFIReference

| Field Name | Description | Type |
|------------|-------------|------|
| `id` | The UUID of the FireFly interface | [`UUID`](simpletypes.md#uuid) |
| `name` | The name of the FireFly interface | `string` |
| `version` | The version of the FireFly interface | `string` |


## FFISerializedEvent

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the event | `string` |
| `description` | A description of the smart contract event | `string` |
| `params` | An array of event parameter/argument definitions | [`FFIParam[]`](#ffiparam) |
| `details` | Additional blockchain specific fields about this event from the original smart contract. Used by the blockchain plugin and for documentation generation. | [`JSONObject`](simpletypes.md#jsonobject) |

## FFIParam

| Field Name | Description | Type |
|------------|-------------|------|
| `name` | The name of the parameter. Note that parameters must be ordered correctly on the FFI, according to the order in the blockchain smart contract | `string` |
| `schema` | FireFly uses an extended subset of JSON Schema to describe parameters, similar to OpenAPI/Swagger. Converters are available for native blockchain interface definitions / type systems - such as an Ethereum ABI. See the documentation for more detail | [`JSONAny`](simpletypes.md#jsonany) |



## ContractListenerOptions

| Field Name | Description | Type |
|------------|-------------|------|
| `firstEvent` | A blockchain specific string, such as a block number, to start listening from. The special strings 'oldest' and 'newest' are supported by all blockchain connectors. Default is 'newest' | `string` |


## ListenerFilter

| Field Name | Description | Type |
|------------|-------------|------|
| `event` | The definition of the event, either provided in-line when creating the listener, or extracted from the referenced FFI | [`FFISerializedEvent`](#ffiserializedevent) |
| `location` | A blockchain specific contract identifier. For example an Ethereum contract address, or a Fabric chaincode name and channel | [`JSONAny`](simpletypes.md#jsonany) |
| `interface` | A reference to an existing FFI, containing pre-registered type information for the event | [`FFIReference`](#ffireference) |
| `signature` | The stringified signature of the event and location, as computed by the blockchain plugin | `string` |


