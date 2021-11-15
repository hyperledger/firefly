package apiserver

import (
	"github.com/hyperledger/firefly/internal/config"
)

const (
	MetricsEnabled = "enabled"
	MetricsPath    = "path"
)

func initMetricsConfPrefix(prefix config.Prefix) {
	prefix.AddKnownKey(MetricsEnabled, true)
	prefix.AddKnownKey(MetricsPath, "/metrics")
}
