package reconcilers

import (
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"reflect"
	crtClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func RegisterReconcilersOrDie(
	mgr manager.Manager,
	client crtClient.Client,
	containerManager *container.ContainerManager,
) {
	var reconcilers []Reconciler

	if viper.GetBool(sharedconfig.EnableEBPFKey) {
		ebpfReconciler, err := NewEBPFReconciler(
			client,
			containerManager,
		)

		if err != nil {
			logrus.WithError(err).Panic("unable to create EBPF reconciler")
		}

		reconcilers = append(reconcilers, ebpfReconciler)
	}

	for _, r := range reconcilers {
		if err := r.SetupWithManager(mgr); err != nil {
			logrus.WithError(err).
				WithField("reconciler", reflect.TypeOf(r).Name()).
				Panicf("unable to set up reconciler: %s", err)
		}
	}
}
