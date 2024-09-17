package main

import (
	"github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/reconcilers"
	"github.com/otterize/network-mapper/src/node-agent/pkg/service"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	signalHandlerCtx := ctrl.SetupSignalHandler()

	service.InitializeService()
	mgr, client := service.CreateControllerRuntimeComponentsOrDie()

	criClient := service.CreateCRIClientOrDie()
	containerManager := container.NewContainerManager(criClient)

	if viper.GetBool(sharedconfig.EnableEBPFKey) {
		ebpf.LoadEBpfPrograms()
	} else {
		logrus.Debug("eBPF programs are disabled")
	}

	reconcilers.RegisterReconcilersOrDie(
		mgr,
		client,
		containerManager,
	)

	if err := mgr.Start(signalHandlerCtx); err != nil {
		logrus.WithError(err).Panic("problem running manager")
	}
}
