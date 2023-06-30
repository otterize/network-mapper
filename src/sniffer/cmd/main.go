package main

import (
	"context"

	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/otterize/network-mapper/src/sniffer/pkg/ipresolver"
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/otterize/network-mapper/src/sniffer/pkg/sniffer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	mapperClient := mapperclient.NewMapperClient(viper.GetString(sharedconfig.MapperApiUrlKey))
	procFsResolver := ipresolver.NewProcFSIPResolver()
	defer procFsResolver.Stop()

	snifferInstance := sniffer.NewSniffer(mapperClient, procFsResolver)
	err := snifferInstance.RunForever(context.Background())
	if err != nil {
		panic(err)
	}
}
