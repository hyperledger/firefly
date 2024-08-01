A contract listener configures FireFly to stream events from the blockchain,
from a specific location on the blockchain, according to a given definition
of the interface for that event.

Check out the [Custom Contracts Tutorial](../../tutorials/custom_contracts/index.md) for
a walk-through of how to set up listeners for the events from your smart contracts.
 
See below for a deep dive into the format of contract listeners and important concepts to understand when managing them. 

###  Multiple filters

From v1.3.1 onwards, a contract listener can be created with multiple filters under a single topic when supported by the connector. 

Before this change each contract listener would only support listening to one specific event from an interface previously defined. A connector specific event signature would be created to easily identify and search for the contact listener for that event. Within that listener, you could optionally specify a location to listen from. Alongside declaring what and where to listen from, you need to specify a `topic` which determines the ordered stream there events are part of. This format is still supported by the API.

Now from v1.3.1, we have extended the contract listeners API to enable users to supply multiple filters where each contains the event to listen to and the optional location. This new feature will allow to better manage contract listeners and group togethere events your application cares about. Each filter will also contain a new connector specific signature which is constructed from the location and the event signature. FireFly will restrict the creation of a contract listener with duplicate filters or supersets. For example, if two filters are listening to the same event but one has specified a location and the other hasn't then the latter will be a superset and already be listening to all the events matching the first filter. Notice that for backwards compatibility the response from the API will populate the `event` and `location` with the contents of the first event filter in the array.

FireFly made a conscious decision to restricts the creation of duplicate contract listeners and would use the values mentioned earlier to create a unique combination of signature + topic + location. If you tried to create the same listener, you would get a 409. This combination is super useful to a developer to asset that there listener exists. When using muliple filters we can no longer rely on the signature of one event and the location of one filters to calculate uniqueness of a contract listener. As part of v1.3.1, we have come up with a new format where we create a new signature format per filter of location + event signature and then concatinate all of these signatures to build the contract listener signature. This contract listener signature containing all the locations and event signtures combined with the topic will guarantee uniqueness of the contract listener. 

### Signature enhancements

As mentioned above, we have introduced a new format for signatures of contract listener and filter signature:
- Each filter signature will be a combination of the location and the specific connector event signature
- Each contract listener signature will be a concatination of all the filter signatures

Furthemore, we noticed that specifically for Ethereum the ABI does not include a differenciation between indexed and non-indexed fields so we included a new section to the Ethereum specific event signature to add the index of the indexed fields on the signature, as such a filter listening to the event `Changed(address indexed from, uint256 value)` at address `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1` will result in `0xa5ea5d0a6b2eaf194716f0cc73981939dca26da1:Changed(address,uint256) [i=0]` where `[i=0]` specifies that the first field is indexed. If there were more indexed fields, it will be a comma separate list of the index of those indexed fields such as `[i=0,2,3]` specifies that fields at index 0, 2, and 3 are indexed fields.


### Formats supported

As described above, there are two formats supported by the API for backwards compatibility with previous releases.

*** Muliple Filters ***

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
    },
  ],
  "options": {
    "firstEvent": "newest"
  },
  "topic": "simple-storage"
}
```

*** One filter with old format ***


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