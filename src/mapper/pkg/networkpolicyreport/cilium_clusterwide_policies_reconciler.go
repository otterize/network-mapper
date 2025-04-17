package networkpolicyreport

import (
	"context"
	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/samber/lo"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/yaml"
)

type CiliumClusterWideNetworkPolicyReconciler struct {
	client.Client
	otterizeCloud cloudclient.CloudClient
}

func NewCiliumClusterWideNetworkPolicyReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient) *CiliumClusterWideNetworkPolicyReconciler {
	return &CiliumClusterWideNetworkPolicyReconciler{
		Client:        client,
		otterizeCloud: otterizeCloudClient,
	}
}

func (r *CiliumClusterWideNetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ciliumv2.CiliumClusterwideNetworkPolicy{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *CiliumClusterWideNetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ciliumClusterWideNetpolList ciliumv2.CiliumClusterwideNetworkPolicyList
	err := r.List(ctx, &ciliumClusterWideNetpolList)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	ciliumClusterWideNetpolsToReport, err := r.convertToNetworkPolicyInputs(ciliumClusterWideNetpolList.Items)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	err = r.otterizeCloud.ReportCiliumClusterWideNetworkPolicies(ctx, ciliumClusterWideNetpolsToReport)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}
	return ctrl.Result{}, nil
}

func (r *CiliumClusterWideNetworkPolicyReconciler) convertToNetworkPolicyInputs(ciliumClusterWideNetpols []ciliumv2.CiliumClusterwideNetworkPolicy) ([]cloudclient.NetworkPolicyInput, error) {
	ciliumClusterWideNetpolsToReport := make([]cloudclient.NetworkPolicyInput, 0)
	for _, ciliumClusterWideNetpol := range ciliumClusterWideNetpols {
		ciliumClusterWideNetpolToReport, err := r.convertToNetworkPolicyInput(ciliumClusterWideNetpol)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		ciliumClusterWideNetpolsToReport = append(ciliumClusterWideNetpolsToReport, ciliumClusterWideNetpolToReport)
	}

	return ciliumClusterWideNetpolsToReport, nil
}

func (r *CiliumClusterWideNetworkPolicyReconciler) convertToNetworkPolicyInput(ciliumClusterWideNetpol ciliumv2.CiliumClusterwideNetworkPolicy) (cloudclient.NetworkPolicyInput, error) {
	ciliumClusterWideNetpol.ObjectMeta = filterObjectMetadata(ciliumClusterWideNetpol.ObjectMeta)
	ciliumClusterWideNetpol.Status = ciliumv2.CiliumNetworkPolicyStatus{}

	yamlString, err := yaml.Marshal(ciliumClusterWideNetpol)
	if err != nil {
		return cloudclient.NetworkPolicyInput{}, errors.Wrap(err)
	}
	return cloudclient.NetworkPolicyInput{
		Name: ciliumClusterWideNetpol.Name,
		Yaml: string(yamlString),
	}, nil
}

func IsCiliumClusterWideInstalledInstalled(ctx context.Context, client client.Client) (bool, error) {
	clusterWideCRDName := "ciliumclusterwidenetworkpolicies.cilium.io"
	crd := apiextensionsv1.CustomResourceDefinition{}
	err := client.Get(ctx, types.NamespacedName{Name: clusterWideCRDName}, &crd)
	if err != nil && !k8serrors.IsNotFound(err) {
		return false, errors.Wrap(err)
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	return true, nil
}
