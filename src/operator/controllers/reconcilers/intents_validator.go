package reconcilers

import (
	"context"
	"fmt"
	otterizev1alpha1 "github.com/otterize/otternose/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IntentsValidatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *IntentsValidatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	intents := &otterizev1alpha1.Intents{}
	err := r.Get(ctx, req.NamespacedName, intents)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, service := range intents.Spec.Services {
		fmt.Println("Intents for service: " + service.Name)
		for _, intent := range service.Calls {
			fmt.Printf("%s has intent to access %s. Intent type: %s\n", service.Name, intent.Server, intent.Type)
		}
	}

	return ctrl.Result{}, nil
}
