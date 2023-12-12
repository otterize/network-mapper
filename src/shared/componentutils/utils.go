package componentutils

import (
	"github.com/otterize/intents-operator/src/shared/otterizecloud/otterizecloudclient"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func SetCloudClientId() {
	// Cloud client ID is set inside the cloud client initialization, this method is meant to be used from components
	// that don't use the cloud client, but still need it for their telemetry data.
	componentinfo.SetGlobalCloudClientId(viper.GetString(otterizecloudclient.ApiClientIdKey))
}

func ExitDueToInitFailure(entry *logrus.Entry, message string) {
	if entry == nil {
		panic(message)
	}

	msg, err := entry.WithField("message", message).String()
	if err != nil {
		// Entry format error, just use the original message
		msg, _ = logrus.WithField("error formatting message", err).WithField("message", message).String()
	}

	// Bugsnag panic synchronously, making sure the massage is sent before exiting
	panic(msg)
}
