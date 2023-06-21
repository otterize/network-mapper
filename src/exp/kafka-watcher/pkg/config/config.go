package config

import (
	"github.com/spf13/viper"
	"time"
)

const (
	KubernetesLogReadMode string = "k8s-logs"
	FileReadMode          string = "file-logs"
)

const (
	KafkaLogReadModeKey          = "kafka-log-read-mode"
	KafkaServersKey              = "kafka-servers"
	KafkaReportIntervalKey       = "kafka-report-interval"
	KafkaReportIntervalDefault   = 10 * time.Second
	KafkaCooldownIntervalKey     = "kafka-cooldown-interval"
	KafkaCooldownIntervalDefault = 10 * time.Second
	KafkaAuthZLogPathKey         = "kafka-authz-log-path"
	KafkaAuthZLogPathDefault     = "/opt/otterize/kafka-watcher/authz.log"
)

func init() {
	viper.SetDefault(KafkaReportIntervalKey, KafkaReportIntervalDefault)
	viper.SetDefault(KafkaServersKey, []string{})
	viper.SetDefault(KafkaCooldownIntervalKey, KafkaCooldownIntervalDefault)
	viper.SetDefault(KafkaAuthZLogPathKey, KafkaAuthZLogPathDefault)
	viper.SetDefault(KafkaLogReadModeKey, KubernetesLogReadMode)
}
