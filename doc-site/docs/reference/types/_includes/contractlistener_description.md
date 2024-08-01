A contract listener configures FireFly to stream events from the blockchain,
from a specific location on the blockchain, according to a given definition
of the interface for that event.

Check out the [Custom Contracts Tutorial](../../tutorials/custom_contracts/index.md) for
a walk-through of how to set up listeners for the events from your smart contracts.

See below for a deep dive into the format of contract listeners and important concepts to understand when managing them.

### Multiple filters

From v1.3.1 onwards, a contract listener can be created with multiple filters under a single topic when supported by the connector.

Before this change, each contract listener would only support listening to one specific event from an interface previously defined. Each listener would be comprised of:

- a reference to a specific blockchain event to listen for
- (optional) a specific location/address to listen from
- a connector-specific signature (generated from the event), which allows you to easily identify and search for the contact listener for an event
- a `topic` which determines the ordered stream that these events are part of

This format is still supported by the API. However, it may not fully guarantee accurate ordering of events coming from multiple listeners, even if they share the same `topic` (such as during cases of blockchain catch-up, or when one listener is created much later than the others).

From v1.3.1, you can supply multiple filters on a single listener. Each filter contains:

- a reference to a specific blockchain event to listen for
- (optional) a specific location/address to listen from
- a connector-specific signature (generated from the event and the location)

In addition to this list of multiple filters, the listener specifies a single `topic` to identify the stream of events. This new feature will allow better management of contract listeners and strong ordering of all of the events your application cares about.

Note: For backwards compatibility, the response from the API will populate top-level `event` and `location` fields with the contents of the first event filter in the array.

### Duplicate filters

FireFly will restrict the creation of a contract listener with duplicate filters or superset filters. For example, if two filters are listening to the same event, but one has specified a location and the other hasn't, then the latter will be a superset, and already be listening to all the events matching the first filter. Creation of duplicate or superset filters will be blocked.

### Duplicate listeners

As of v1.3.1, each filter on a listener includes a signature generated from the filter location + event, and the listener concatenates all of these signatures to build the overall contract listener signature. This contract listener signature - containing all the locations and event signatures combined with the listener topic - will guarantee uniqueness of the contract listener. If you tried to create the same listener again, you would receive HTTP 409. This combination can allow a developer to assert that their listener exists, without the risk of creating duplicates.

Note: Prior to v1.3.1, FireFly would detect duplicates simply by requiring a unique combination of signature + topic + location for each listener. When using muliple filters, we cannot rely on the signature of one event and the location of one filter to calculate uniqueness of a contract listener, hence the need for the more sophisticated uniqueness checks described here.

### Signature enhancements

As mentioned above, we have introduced a new format for signatures of contract listener and filter signature:

- Each filter signature will be a combination of the location and the specific connector event signature
- Each contract listener signature will be a concatenation of all the filter signatures

Furthermore, because Ethereum ABI does not include a differentiation between indexed and non-indexed fields, we have included a new section in the Ethereum-specific event signature to add the index of the indexed fields. As such, a filter listening to the event `Changed(address indexed from, uint256 value)` at address `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1` will result in `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1:Changed(address,uint256) [i=0]` where `[i=0]` specifies that the first field is indexed. If there were more indexed fields, it will be a comma-separated list of the index of those indexed fields such as `[i=0,2,3]` (specifies that fields at index 0, 2, and 3 are indexed fields).

### Formats supported

As described above, there are two input formats supported by the API for backwards compatibility with previous releases.

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
