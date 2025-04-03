package networkpolicyreport

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/yaml"
)

type NetworkPolicyReconciler struct {
	client.Client
	otterizeCloud cloudclient.CloudClient
}

func NewNetworkPolicyReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient) *NetworkPolicyReconciler {
	return &NetworkPolicyReconciler{
		Client:        client,
		otterizeCloud: otterizeCloudClient,
	}
}

func (r *NetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.NetworkPolicy{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *NetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	namespace := req.Namespace
	var netpolList networkingv1.NetworkPolicyList
	err := r.List(ctx, &netpolList, client.InNamespace(namespace))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	netpolsToReport, err := r.convertToNetworkPolicyInputs(netpolList.Items)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	err = r.otterizeCloud.ReportNetworkPolicies(ctx, namespace, netpolsToReport)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}
	return ctrl.Result{}, nil
}

func (r *NetworkPolicyReconciler) convertToNetworkPolicyInputs(netpols []networkingv1.NetworkPolicy) ([]cloudclient.NetworkPolicyInput, error) {
	netpolsToReport := make([]cloudclient.NetworkPolicyInput, 0)
	for _, netpol := range netpols {
		netpolToReport, err := r.convertToNetworkPolicyInput(netpol)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		netpolsToReport = append(netpolsToReport, netpolToReport)
	}
	return netpolsToReport, nil
}

func (r *NetworkPolicyReconciler) convertToNetworkPolicyInput(netpol networkingv1.NetworkPolicy) (cloudclient.NetworkPolicyInput, error) {
	netpol.ManagedFields = nil
	netpol.OwnerReferences = nil
	netpol.Finalizers = nil
	yamlBytes, err := yaml.Marshal(netpol)
	if err != nil {
		return cloudclient.NetworkPolicyInput{}, errors.Wrap(err)
	}
	return cloudclient.NetworkPolicyInput{
		Name: netpol.Name,
		Yaml: string(yamlBytes),
	}, nil
}
