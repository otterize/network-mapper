package service

import (
	"github.com/bombsimon/logrusr/v3"
	"github.com/otterize/network-mapper/src/shared/componentutils"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

func InitializeService() {
	logrus.SetLevel(logrus.InfoLevel)

	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	ctrl.SetLogger(logrusr.New(logrus.StandardLogger()))

	componentutils.RegisterPanicHandlers()
}
