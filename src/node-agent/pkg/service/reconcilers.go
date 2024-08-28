package service

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/reconcilers"
	"github.com/sirupsen/logrus"
	"reflect"
	crtClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func RegisterReconcilersOrDie(
	mgr manager.Manager,
	client crtClient.Client,
	containerManager *container.ContainerManager,
	finder *kubefinder.KubeFinder,
) {
	ebpfReconciler, err := reconcilers.NewEBPFReconciler(
		client,
		containerManager,
		finder,
	)

	if err != nil {
		logrus.WithError(err).Panic("unable to create EBPF reconciler")
	}

	reconcilersToRegister := []reconcilers.Reconciler{
		ebpfReconciler,
	}

	for _, r := range reconcilersToRegister {
		if err := r.SetupWithManager(mgr); err != nil {
			logrus.WithError(err).
				WithField("reconciler", reflect.TypeOf(r).Name()).
				Panicf("unable to set up reconciler: %s", err)
		}
	}
}
