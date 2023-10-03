package metricexporter

import (
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
)

type Config struct {
	ExportInterval time.Duration
}

func ConfigFromViper() Config {
	return Config{
		ExportInterval: time.Duration(viper.GetInt(config.OTelExportIntervalSecondsKey)) * time.Second,
	}
}
