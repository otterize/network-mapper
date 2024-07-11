package reconcilers

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr manager.Manager) error
}
