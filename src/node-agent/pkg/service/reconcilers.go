package service

import (
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
//bpfmanClient bpfmanclient.BpfmanClient,
	containerManager *container.ContainerManager,
) {
	reconcilersToRegister := []reconcilers.Reconciler{
		reconcilers.NewEBPFReconciler(
			client,
			//bpfmanClient,
			containerManager,
		),
	}

	for _, r := range reconcilersToRegister {
		if err := r.SetupWithManager(mgr); err != nil {
			logrus.WithError(err).
				WithField("reconciler", reflect.TypeOf(r).Name()).
				Panicf("unable to set up reconciler: %s", err)
		}
	}
}
