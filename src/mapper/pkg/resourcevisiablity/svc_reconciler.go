package resourcevisiablity

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
)

type ServiceReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	otterizeCloud cloudclient.CloudClient
	kubeFinder    *kubefinder.KubeFinder
}

func NewServiceReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient, kubeFinder *kubefinder.KubeFinder) *ServiceReconciler {
	return &ServiceReconciler{
		Client:        client,
		otterizeCloud: otterizeCloudClient,
		kubeFinder:    kubeFinder,
	}
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	// We subscribe to the Endpoints resource, to make sure that any changes in pods that is selected by a service
	// will trigger a reconciliation of the namespace.
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *ServiceReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	namespace := req.Namespace
	var ServiceList corev1.ServiceList
	err := r.List(ctx, &ServiceList, client.InNamespace(namespace))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	servicesToReport, err := r.convertToCloudServices(ctx, ServiceList.Items)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	err = r.otterizeCloud.ReportK8sServices(ctx, namespace, servicesToReport)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) convertToCloudServices(ctx context.Context, services []corev1.Service) ([]cloudclient.K8sServiceInput, error) {
	cloudServices := make([]cloudclient.K8sServiceInput, 0)
	for _, service := range services {
		cloudService, _, err := r.convertToCloudService(ctx, service)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		cloudServices = append(cloudServices, cloudService)
	}

	return cloudServices, nil
}

func (r *ServiceReconciler) convertToCloudService(ctx context.Context, service corev1.Service) (cloudclient.K8sServiceInput, bool, error) {
	identity, found, err := r.kubeFinder.ResolveOtterizeIdentityForService(ctx, &service, time.Now())
	if err != nil {
		return cloudclient.K8sServiceInput{}, false, errors.Wrap(err)
	}

	if !found {
		return cloudclient.K8sServiceInput{}, false, nil
	}

	serviceInput, err := convertServiceResource(service)
	if err != nil {
		return cloudclient.K8sServiceInput{}, false, errors.Wrap(err)
	}

	cloudService := cloudclient.K8sServiceInput{
		OtterizeServer: identity.Name,
		ResourceName:   service.Name,
		Namespace:      identity.Namespace,
		Service:        serviceInput,
	}

	return cloudService, false, nil
}
