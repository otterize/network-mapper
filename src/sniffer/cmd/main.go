package main

import (
	"context"
	"github.com/otterize/otternose/sniffer/pkg/config"
	"github.com/otterize/otternose/sniffer/pkg/sniffer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if viper.GetBool(config.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}

	snifferInstance := sniffer.NewSniffer()
	err := snifferInstance.RunForever(context.Background())
	if err != nil {
		panic(err)
	}
}
