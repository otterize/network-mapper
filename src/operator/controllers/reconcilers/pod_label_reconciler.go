package reconcilers

import (
	"context"
	otterizev1alpha1 "github.com/otterize/otternose/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodLabelsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PodLabelsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	intents := &otterizev1alpha1.Intents{}
	err := r.Get(ctx, req.NamespacedName, intents)
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO: Label pods here by intents

	return ctrl.Result{}, nil
}
