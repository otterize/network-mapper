package clouduploader

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	UploadInterval time.Duration
}

func ConfigFromViper() Config {
	return Config{
		UploadInterval: time.Duration(viper.GetInt(config.UploadIntervalSecondsKey)) * time.Second,
	}
}
