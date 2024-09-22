package main

import (
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/reconcilers"
	"github.com/otterize/network-mapper/src/node-agent/pkg/service"
	"github.com/otterize/network-mapper/src/shared/cloudclient"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	signalHandlerCtx := ctrl.SetupSignalHandler()

	service.InitializeService()
	mgr, client := service.CreateControllerRuntimeComponentsOrDie()

	criClient := service.CreateCRIClientOrDie()
	containerManager := container.NewContainerManager(criClient)

	cloudClient, cloudEnabled, err := cloudclient.NewClient(signalHandlerCtx)
	if err != nil {
		logrus.WithError(err).Panic("Failed to initialize cloud client")
	}
	if !cloudEnabled {
		logrus.WithError(err).Panic("Cloud client is not enabled")
	}

	reconcilers.RegisterReconcilersOrDie(
		mgr,
		client,
		cloudClient,
		containerManager,
	)

	if err := mgr.Start(signalHandlerCtx); err != nil {
		logrus.WithError(err).Panic("problem running manager")
	}
}
