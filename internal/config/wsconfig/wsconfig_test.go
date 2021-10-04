package wsconfig

import (
	"testing"
	"time"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/stretchr/testify/assert"
)

var utConfPrefix = config.NewPluginConfig("ws")

func resetConf() {
	config.Reset()
	InitPrefix(utConfPrefix)
}

func TestWSConfigGeneration(t *testing.T) {
	resetConf()

	utConfPrefix.Set(restclient.HTTPConfigURL, "http://test:12345")
	utConfPrefix.Set(restclient.HTTPConfigHeaders, map[string]interface{}{
		"custom-header": "custom value",
	})
	utConfPrefix.Set(restclient.HTTPConfigAuthUsername, "user")
	utConfPrefix.Set(restclient.HTTPConfigAuthPassword, "pass")
	utConfPrefix.Set(restclient.HTTPConfigRetryInitDelay, 1)
	utConfPrefix.Set(restclient.HTTPConfigRetryMaxDelay, 1)
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
