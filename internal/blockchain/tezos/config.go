package tezos

import (
	"github.com/hyperledger/firefly-common/pkg/config"
	"github.com/hyperledger/firefly-common/pkg/ffresty"
	"github.com/hyperledger/firefly-common/pkg/wsclient"
)

const (
	defaultBatchSize    = 50
	defaultBatchTimeout = 500
	defaultPrefixShort  = "fly"
	defaultPrefixLong   = "firefly"
	defaultFromBlock    = "0"

	defaultAddressResolverMethod        = "GET"
	defaultAddressResolverResponseField = "address"
)

const (
	// TezosconnectConfigKey is a sub-key in the config to contain all the tezosconnect specific config
	TezosconnectConfigKey = "tezosconnect"
	// TezosconnectConfigTopic is the websocket listen topic that the node should register on, which is important if there are multiple
	// nodes using a single tezosconnect
	TezosconnectConfigTopic = "topic"
	// TezosconnectConfigBatchSize is the batch size to configure on event streams, when auto-defining them
	TezosconnectConfigBatchSize = "batchSize"
	// TezosconnectConfigBatchTimeout is the batch timeout to configure on event streams, when auto-defining them
	TezosconnectConfigBatchTimeout = "batchTimeout"
	// TezosconnectPrefixShort is used in the query string in requests to tezosconnect
	TezosconnectPrefixShort = "prefixShort"
	// TezosconnectPrefixLong is used in HTTP headers in requests to tezosconnect
	TezosconnectPrefixLong = "prefixLong"
	// TezosconnectConfigInstanceDeprecated is the tezos address of the FireFly contract
	TezosconnectConfigInstanceDeprecated = "instance"
	// TezosconnectConfigFromBlockDeprecated is the configuration of the first block to listen to when creating the listener for the FireFly contract
	TezosconnectConfigFromBlockDeprecated = "fromBlock"

	// AddressResolverConfigKey is a sub-key in the config to contain an address resolver config.
	AddressResolverConfigKey = "addressResolver"
	// AddressResolverAlwaysResolve causes the address resolve to be invoked on every API call that resolves an address, regardless of whether the input conforms to an 0x address, and disables any caching
	AddressResolverAlwaysResolve = "alwaysResolve"
	// AddressResolverRetainOriginal when true the original pre-resolved string is retained after the lookup, and passed down to Tezosconnect as the from address
	AddressResolverRetainOriginal = "retainOriginal"
	// AddressResolverMethod the HTTP method to use to call the address resolver (default GET)
	AddressResolverMethod = "method"
	// AddressResolverURLTemplate the URL go template string to use when calling the address resolver - a ".intent" string can be used in the go template
	AddressResolverURLTemplate = "urlTemplate"
	// AddressResolverBodyTemplate the body go template string to use when calling the address resolver - a ".intent" string can be used in the go template
	AddressResolverBodyTemplate = "bodyTemplate"
	// AddressResolverResponseField the name of a JSON field that is provided in the response, that contains the tezos address (default "address")
	AddressResolverResponseField = "responseField"

	// FFTMConfigKey is a sub-key in the config that optionally contains FireFly transaction connection information
	FFTMConfigKey = "fftm"
)

func (t *Tezos) InitConfig(config config.Section) {
	t.tezosconnectConf = config.SubSection(TezosconnectConfigKey)
	wsclient.InitConfig(t.tezosconnectConf)
	t.tezosconnectConf.AddKnownKey(TezosconnectConfigTopic)
	t.tezosconnectConf.AddKnownKey(TezosconnectConfigBatchSize, defaultBatchSize)
	t.tezosconnectConf.AddKnownKey(TezosconnectConfigBatchTimeout, defaultBatchTimeout)
	t.tezosconnectConf.AddKnownKey(TezosconnectPrefixShort, defaultPrefixShort)
	t.tezosconnectConf.AddKnownKey(TezosconnectPrefixLong, defaultPrefixLong)
	t.tezosconnectConf.AddKnownKey(TezosconnectConfigInstanceDeprecated)
	t.tezosconnectConf.AddKnownKey(TezosconnectConfigFromBlockDeprecated, defaultFromBlock)

	fftmConf := config.SubSection(FFTMConfigKey)
	ffresty.InitConfig(fftmConf)

	addressResolverConf := config.SubSection(AddressResolverConfigKey)
	ffresty.InitConfig(addressResolverConf)
	addressResolverConf.AddKnownKey(AddressResolverAlwaysResolve)
	addressResolverConf.AddKnownKey(AddressResolverRetainOriginal)
	addressResolverConf.AddKnownKey(AddressResolverMethod, defaultAddressResolverMethod)
	addressResolverConf.AddKnownKey(AddressResolverURLTemplate)
	addressResolverConf.AddKnownKey(AddressResolverBodyTemplate)
	addressResolverConf.AddKnownKey(AddressResolverResponseField, defaultAddressResolverResponseField)
}
