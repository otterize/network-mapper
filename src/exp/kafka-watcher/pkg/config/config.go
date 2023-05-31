package config

import (
	"github.com/spf13/viper"
	"time"
)

const (
	KafkaCooldownIntervalKey     = "kafka-cooldown-interval"
	KafkaCooldownIntervalDefault = 10 * time.Second
)

func init() {
	viper.SetDefault(KafkaCooldownIntervalKey, KafkaCooldownIntervalDefault)
}
