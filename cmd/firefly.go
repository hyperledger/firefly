// Copyright 2019 Kaleido

package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Execute is called by the main method of the package
func Execute() error {

	log := logrus.WithField("pid", os.Getpid())
	log.Infof("Project Firefly")
	log.Infof("© Copyright 2021 Kaleido, Inc.")

	viper.SetConfigName("firefly")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/firefly/")
	viper.AddConfigPath("$HOME/.firefly")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil

}
