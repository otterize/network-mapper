package main

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/service"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	signalHandlerCtx := ctrl.SetupSignalHandler()

	service.InitializeService()
	mgr, client := service.CreateControllerRuntimeComponentsOrDie()

	criClient := service.CreateCRIClientOrDie()
	containerManager := container.NewContainerManager(criClient)

	finder, err := kubefinder.NewKubeFinder(signalHandlerCtx, mgr)

	if err != nil {
		logrus.WithError(err).Panic("unable to create kube finder")
	}

	service.RegisterReconcilersOrDie(
		mgr,
		client,
		containerManager,
		finder,
	)

	if err := mgr.Start(signalHandlerCtx); err != nil {
		logrus.WithError(err).Panic("problem running manager")
	}
}