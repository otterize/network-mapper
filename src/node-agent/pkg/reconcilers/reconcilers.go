package reconcilers

import (
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/shared/cloudclient"
	"github.com/sirupsen/logrus"
	"reflect"
	crtClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func RegisterReconcilersOrDie(
	mgr manager.Manager,
	client crtClient.Client,
	cloudClient cloudclient.CloudClient,
	containerManager *container.ContainerManager,
) {
	ebpfReconciler, err := NewEBPFReconciler(
		client,
		cloudClient,
		containerManager,
	)

	if err != nil {
		logrus.WithError(err).Panic("unable to create EBPF reconciler")
	}

	reconcilersToRegister := []Reconciler{
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
