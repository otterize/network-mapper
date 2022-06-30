package reconcilers

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

type PodsReconciler struct {
	Client      client.Client
	ipToPodInfo *sync.Map
}

func NewPodsReconciler(client client.Client) *PodsReconciler {
	return &PodsReconciler{Client: client, ipToPodInfo: &sync.Map{}}
}

func (r *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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
		r.ipToPodInfo.Store(ip.IP, pod)
	}
	return reconcile.Result{}, nil
}

func (r *PodsReconciler) ResolveIpToPod(ip string) (*v1.Pod, bool) {
	pod, ok := r.ipToPodInfo.Load(ip)
	if !ok {
		return &v1.Pod{}, false
	}
	return pod.(*v1.Pod), ok
}

func (r *PodsReconciler) ResolveLabelSelectorToPodsInfo(ctx context.Context, selector client.MatchingLabels) ([]v1.Pod, error) {
	pods := v1.PodList{}
	err := r.Client.List(ctx, &pods, selector)
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (r *PodsReconciler) Register(mgr manager.Manager) error {
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
