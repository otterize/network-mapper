package resourcevisibility

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/network-mapper/src/shared/cloudclient"
	"github.com/samber/lo"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type IngressReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	otterizeCloud cloudclient.CloudClient
}

func NewIngressReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient) *IngressReconciler {
	return &IngressReconciler{
		Client:        client,
		otterizeCloud: otterizeCloudClient,
	}
}

func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *IngressReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	namespace := req.Namespace
	var IngressList networkingv1.IngressList
	err := r.List(ctx, &IngressList, client.InNamespace(namespace))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	ingressesToReport, err := r.convertToCloudIngresses(IngressList.Items)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	err = r.otterizeCloud.ReportK8sIngresses(ctx, namespace, ingressesToReport)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}
	return ctrl.Result{}, nil
}

func (r *IngressReconciler) convertToCloudIngresses(ingresses []networkingv1.Ingress) ([]cloudclient.K8sIngressInput, error) {
	ingressesToReport := make([]cloudclient.K8sIngressInput, 0)
	for _, ingress := range ingresses {
		ingressInput, ok, err := convertIngressResource(ingress)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if !ok {
			continue
		}

		ingressesToReport = append(ingressesToReport, cloudclient.K8sIngressInput{
			Namespace: ingress.Namespace,
			Name:      ingress.Name,
			Ingress:   ingressInput,
		})
	}
	return ingressesToReport, nil
}
