package componentutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/otterizecloud/otterizecloudclient"
	"github.com/otterize/intents-operator/src/shared/telemetries/componentinfo"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"time"
)

func SetCloudClientId() {
	// Cloud client ID is set inside the cloud client initialization, this method is meant to be used from components
	// that don't use the cloud client, but still need it for their telemetry data.
	componentinfo.SetGlobalCloudClientId(viper.GetString(otterizecloudclient.ApiClientIdKey))
}

func WaitAndSetContextId(ctx context.Context) error {
	interval := viper.GetDuration(sharedconfig.ReadContextIdIntervalKey)

	for {
		ok, err := setContextId()
		if err != nil {
			logrus.WithError(err).Error("Error when setting context id")
			return err
		}
		if ok {
			logrus.Info("Context id set successfully")
			return nil
		}

		select {
		case <-ctx.Done():
			logrus.Info("Context canceled, exiting context id setting loop")
			return nil
		case <-time.After(interval):
			logrus.Info("Context id not set yet, waiting")
		}
	}
}

func setContextId() (bool, error) {
	dirPath := viper.GetString(sharedconfig.ComponentMetadataConfigmapMountPathKey)
	fileName := viper.GetString(sharedconfig.ComponentMetadataContextIdKeyKey)
	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	res, err := os.ReadFile(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	componentinfo.SetGlobalContextId(string(res))
	return true, nil
}
