package reconcilers

import (
	"context"
	"errors"
	otterizev1alpha1 "github.com/otterize/otternose/api/v1alpha1"
	"github.com/sirupsen/logrus"
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
		logrus.Debugln("Intents for service: " + service.Name)
		for _, intent := range service.Calls {
			logrus.Debugf("%s has intent to access %s. Intent type: %s\n", service.Name, intent.Server, intent.Type)
			if err := validateIntent(intent); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func validateIntent(intent otterizev1alpha1.Intent) error {
	if intent.Type == otterizev1alpha1.IntentTypeKafka {
		if intent.HTTPResources != nil {
			return errors.New("invalid intent format. type 'Kafka' cannot contain HTTP resources")
		}
	}

	if intent.Type == otterizev1alpha1.IntentTypeHTTP {
		if intent.Topics != nil {
			return errors.New("invalid intent format. type 'HTTP' cannot contain kafka topics")
		}
	}

	return nil
}
