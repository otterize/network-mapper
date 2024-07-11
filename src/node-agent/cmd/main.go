package main

import (
	"github.com/otterize/network-mapper/src/node-agent/pkg/service"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	signalHandlerCtx := ctrl.SetupSignalHandler()

	service.InitializeService()
	mgr, client := service.CreateControllerRuntimeComponentsOrDie()

	bpfmanClient := service.ConnectToBpfmanOrDie(signalHandlerCtx)

	service.RegisterReconcilersOrDie(
		mgr,
		client,
		bpfmanClient,
	)

	if err := mgr.Start(signalHandlerCtx); err != nil {
		logrus.WithError(err).Panic("problem running manager")
	}
}
