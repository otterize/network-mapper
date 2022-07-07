package reconcilers

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	Client         client.Client
	ipToPod        *sync.Map
	podToOwnerName *sync.Map
}

func NewPodsReconciler(client client.Client) *PodsReconciler {
	return &PodsReconciler{Client: client, ipToPod: &sync.Map{}}
}

func (r *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	pod := &coreV1.Pod{}
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
		r.ipToPod.Store(ip.IP, pod)
	}

	return reconcile.Result{}, nil
}

func (r *PodsReconciler) ResolveIpToPod(ip string) (*coreV1.Pod, bool) {
	pod, ok := r.ipToPod.Load(ip)
	if !ok {
		return &coreV1.Pod{}, false
	}
	return pod.(*coreV1.Pod), ok
}

func (r *PodsReconciler) ResolvePodToOwnerName(ctx context.Context, pod *coreV1.Pod) (string, error) {
	for _, owner := range pod.OwnerReferences {
		namespacedName := types.NamespacedName{Name: owner.Name, Namespace: pod.Namespace}
		switch owner.Kind {
		case "ReplicaSet":
			rs := &appsV1.ReplicaSet{}
			err := r.Client.Get(ctx, namespacedName, rs)
			if err != nil {
				return "", err
			}
			return rs.OwnerReferences[0].Name, nil
		case "DaemonSet":
			ds := &appsV1.DaemonSet{}
			err := r.Client.Get(ctx, namespacedName, ds)
			if err != nil {
				return "", err
			}
			return ds.Name, nil
		default:
			logrus.Infof("Unknown owner kind %s for pod %s", owner.Kind, pod.Name)
		}
	}
	return "", fmt.Errorf("pod %s has no owner", pod.Name)
}

func (r *PodsReconciler) Register(mgr manager.Manager) error {
	podsController, err := controller.New("pods-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("unable to set up pods controller: %w", err)
	}

	err = podsController.Watch(&source.Kind{Type: &coreV1.Pod{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("unable to watch Pods: %w", err)
	}

	return nil
}
