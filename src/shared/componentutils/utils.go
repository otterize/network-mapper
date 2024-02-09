package componentutils

import (
	"github.com/otterize/intents-operator/src/shared/otterizecloud/otterizecloudclient"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	"github.com/spf13/viper"
)

func SetCloudClientId() {
	// Cloud client ID is set inside the cloud client initialization, this method is meant to be used from components
	// that don't use the cloud client, but still need it for their telemetry data.
	componentinfo.SetGlobalCloudClientId(viper.GetString(otterizecloudclient.ApiClientIdKey))
}
