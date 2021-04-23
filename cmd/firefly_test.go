// Copyright 2019 Kaleido

package cmd

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestExecMissingConfig(t *testing.T) {
	viper.Reset()
	err := Execute()
	assert.Regexp(t, "Not Found", err.Error())
}

func TestExecOk(t *testing.T) {
	os.Chdir("../test/config")
	err := Execute()
	assert.NoError(t, err)
}
