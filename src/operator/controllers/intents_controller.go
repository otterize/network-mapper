/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	reconcilers "github.com/otterize/otternose/controllers/reconcilers"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	otterizev1alpha1 "github.com/otterize/otternose/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IntentsReconciler reconciles a Intents object
type IntentsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=otterize.com,resources=intents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=otterize.com,resources=intents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=otterize.com,resources=intents/finalizers,verbs=update
//+kubebuilder:rbac:groups=otterize.com,resources=pods,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Intents object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *IntentsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reconcilersList, err := buildReconcilersList()
	if err != nil {
		return ctrl.Result{}, err
	}

	logrus.Infoln("## Starting new Otterize reconciliation cycle ##")
	for _, r := range reconcilersList {
		logrus.Infof("Starting cycle for %T\n", r)
		res, err := r.Reconcile(ctx, req)
		if res.Requeue == true || err != nil {
			return res, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IntentsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&otterizev1alpha1.Intents{}).
		Complete(r)
}

func buildReconcilersList() ([]reconcile.Reconciler, error) {
	l := make([]reconcile.Reconciler, 0)

	l = append(l, &reconcilers.IntentsValidatorReconciler{})

	return l, nil
}
