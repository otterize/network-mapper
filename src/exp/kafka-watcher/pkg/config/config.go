package config

import (
	"github.com/spf13/viper"
	"time"
)

const (
	KafkaServersKey              = "kafka-servers"
	KafkaReportIntervalKey       = "kafka-report-interval"
	KafkaReportIntervalDefault   = 10 * time.Second
	KafkaCooldownIntervalKey     = "kafka-cooldown-interval"
	KafkaCooldownIntervalDefault = 10 * time.Second
)

func init() {
	viper.SetDefault(KafkaReportIntervalKey, KafkaReportIntervalDefault)
	viper.SetDefault(KafkaCooldownIntervalKey, KafkaCooldownIntervalDefault)
	viper.SetDefault(KafkaServersKey, []string{})
}
