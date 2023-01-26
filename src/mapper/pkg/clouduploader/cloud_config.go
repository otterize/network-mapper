package clouduploader

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
)

type Config struct {
	UploadInterval int
}

func ConfigFromViper() Config {
	return Config{
		UploadInterval: viper.GetInt(config.UploadIntervalSecondsKey),
	}
}
