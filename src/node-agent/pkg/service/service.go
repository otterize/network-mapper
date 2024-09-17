package service

import (
	"github.com/bombsimon/logrusr/v3"
	"github.com/otterize/intents-operator/src/shared"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// InitializeService sets up the basic infrastructure for the service
func InitializeService() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.Infof("Running in debug mode")
		logrus.SetLevel(logrus.DebugLevel)
	}

	ctrl.SetLogger(logrusr.New(logrus.StandardLogger()))

	shared.RegisterPanicHandlers()
}
