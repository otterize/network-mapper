package clouduploader

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
)

type Config struct {
	ClientId       string
	Secret         string
	apiAddress     string
	UploadInterval int
}

func ConfigFromViper() Config {
	return Config{
		Secret:         viper.GetString(config.ClientSecretKey),
		ClientId:       viper.GetString(config.ClientIDKey),
		apiAddress:     viper.GetString(config.CloudApiAddrKey),
		UploadInterval: viper.GetInt(config.UploadIntervalSecondsKey),
	}
}

func (c *Config) IsCloudUploadEnabled() bool {
	return c.ClientId != "" && c.Secret != ""
}
