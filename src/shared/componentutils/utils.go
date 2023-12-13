package componentutils

import (
	"context"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/otterize/intents-operator/src/shared/otterizecloud/otterizecloudclient"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

func SetCloudClientId() {
	// Cloud client ID is set inside the cloud client initialization, this method is meant to be used from components
	// that don't use the cloud client, but still need it for their telemetry data.
	componentinfo.SetGlobalCloudClientId(viper.GetString(otterizecloudclient.ApiClientIdKey))
}

func ExitDueToInitFailure(entry *logrus.Entry, message string) {
	triggerBugsnagSync(entry, message)
	if entry != nil {
		entry.Error(message)
	} else {
		logrus.Error(message)
	}
	os.Exit(1)
}

func triggerBugsnagSync(entry *logrus.Entry, message string) {
	// The only way to trigger a synchronous bugsnag report is to call panic, and catch with AutoNotify
	// This function triggers a panic and recovers from it to avoid printing the panic stack trace to the user
	defer func() {
		_ = recover()
	}()

	defer bugsnag.AutoNotify(context.Background())
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
