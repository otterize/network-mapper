package operators

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
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

type PodInfo struct {
	Namespace string
	Name      string
}

type PodsOperator struct {
	Client      client.Client
	ipToPodInfo *sync.Map
}

func NewPodsOperator(client client.Client) *PodsOperator {
	return &PodsOperator{Client: client, ipToPodInfo: &sync.Map{}}
}

func (r *PodsOperator) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	pod := &v1.Pod{}
	err := r.Client.Get(ctx, request.NamespacedName, pod)
	if errors.IsNotFound(err) {
		logrus.Debug("Pod was deleted")
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch Pod: %+v", err)
	}

	logrus.WithFields(logrus.Fields{"name": pod.Name, "namespace": pod.Namespace}).Debug("Reconciling Pod")
	for _, ip := range pod.Status.PodIPs {
		r.ipToPodInfo.Store(ip.IP, PodInfo{
			Namespace: pod.Namespace,
			Name:      pod.Name,
		})
	}

	return reconcile.Result{}, nil
}

func (r *PodsOperator) ResolveIpToPodInfo(ip string) (PodInfo, bool) {
	podInfo, ok := r.ipToPodInfo.Load(ip)
	if !ok {
		return PodInfo{}, false
	}
	return podInfo.(PodInfo), ok
}

func (r *PodsOperator) Register(mgr manager.Manager) error {
	podsController, err := controller.New("pods-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("unable to set up pods controller: %w", err)
	}

	err = podsController.Watch(&source.Kind{Type: &v1.Pod{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("unable to watch Pods: %w", err)
	}
	return nil
}
