package wsconfig

import (
	"testing"
	"time"

	"github.com/hyperledger/firefly/internal/coreconfig"
	"github.com/hyperledger/firefly/pkg/config"
	"github.com/hyperledger/firefly/pkg/ffresty"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("ws")

func resetConf() {
	coreconfig.Reset()
	InitPrefix(utConfPrefix)
}

func TestWSConfigGeneration(t *testing.T) {
	resetConf()

	utConfPrefix.Set(ffresty.HTTPConfigURL, "http://test:12345")
	utConfPrefix.Set(ffresty.HTTPConfigHeaders, map[string]interface{}{
		"custom-header": "custom value",
	})
	utConfPrefix.Set(ffresty.HTTPConfigAuthUsername, "user")
	utConfPrefix.Set(ffresty.HTTPConfigAuthPassword, "pass")
	utConfPrefix.Set(ffresty.HTTPConfigRetryInitDelay, 1)
	utConfPrefix.Set(ffresty.HTTPConfigRetryMaxDelay, 1)
	utConfPrefix.Set(WSConfigKeyReadBufferSize, 1024)
	utConfPrefix.Set(WSConfigKeyWriteBufferSize, 1024)
	utConfPrefix.Set(WSConfigKeyInitialConnectAttempts, 1)
	utConfPrefix.Set(WSConfigKeyPath, "/websocket")

	wsConfig := GenerateConfigFromPrefix(utConfPrefix)

	assert.Equal(t, "http://test:12345", wsConfig.HTTPURL)
	assert.Equal(t, "user", wsConfig.AuthUsername)
	assert.Equal(t, "pass", wsConfig.AuthPassword)
	assert.Equal(t, time.Duration(1000000), wsConfig.InitialDelay)
	assert.Equal(t, time.Duration(1000000), wsConfig.MaximumDelay)
	assert.Equal(t, 1, wsConfig.InitialConnectAttempts)
	assert.Equal(t, "/websocket", wsConfig.WSKeyPath)
	assert.Equal(t, "custom value", wsConfig.HTTPHeaders.GetString("custom-header"))
	assert.Equal(t, 1024, wsConfig.ReadBufferSize)
	assert.Equal(t, 1024, wsConfig.WriteBufferSize)
}
