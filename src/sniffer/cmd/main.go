package main

import (
	"context"
	"github.com/otterize/network-mapper/src/sniffer/pkg/client"
	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/sniffer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if viper.GetBool(config.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	mapperClient := client.NewMapperClient(viper.GetString(config.MapperApiUrlKey))
	snifferInstance := sniffer.NewSniffer(mapperClient)
	err := snifferInstance.RunForever(context.Background())
	if err != nil {
		panic(err)
	}
}
