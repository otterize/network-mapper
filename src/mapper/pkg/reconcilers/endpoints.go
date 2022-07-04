package reconcilers

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/mapper/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type EndpointsReconciler struct {
	Client           client.Client
	serviceNameToIps *sync.Map
}

func NewEndpointsReconciler(client client.Client) *EndpointsReconciler {
	return &EndpointsReconciler{
		Client:           client,
		serviceNameToIps: &sync.Map{},
	}
}

func (r *EndpointsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	endpoint := &v1.Endpoints{}
	err := r.Client.Get(ctx, request.NamespacedName, endpoint)
	if errors.IsNotFound(err) {
		logrus.Debug("endpoint was deleted")
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch endpoint: %+v", err)
	}

	logrus.WithFields(logrus.Fields{"name": endpoint.Name, "namespace": endpoint.Namespace}).Debug("Reconciling endpoint")
	ips := make([]string, 0)
	for _, subset := range endpoint.Subsets {
		for _, address := range subset.Addresses {
			ips = append(ips, address.IP)
		}
	}
	r.serviceNameToIps.Store(fmt.Sprintf("%s.%s.svc.%s", endpoint.Name, endpoint.Namespace, viper.GetString(config.ClusterDomainKey)), ips)

	return reconcile.Result{}, nil
}

func (r *EndpointsReconciler) ResolveServiceAddressToIps(address string) ([]string, bool) {
	ips, ok := r.serviceNameToIps.Load(address)
	if !ok {
		return nil, false
	}
	return ips.([]string), true
}

func (r *EndpointsReconciler) Register(mgr manager.Manager) error {
	endpointsController, err := controller.New("endpoints-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("unable to set up endpoints controller: %w", err)
	}

	err = endpointsController.Watch(&source.Kind{Type: &v1.Service{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("unable to watch Endpoints: %w", err)
	}
	return nil
}
