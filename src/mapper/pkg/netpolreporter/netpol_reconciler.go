package netpolreporter

import (
	"context"
	"github.com/amit7itz/goset"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
)

const (
	listPodsForPolicyRetryDelay = 5 * time.Second
)

type NetworkPolicyUploaderReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	serviceIdResolver *serviceidresolver.Resolver
	otterizeClient    cloudclient.CloudClient
}

func NewNetworkPolicyUploaderReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	otterizeClient cloudclient.CloudClient,
) *NetworkPolicyUploaderReconciler {
	return &NetworkPolicyUploaderReconciler{
		Client:            client,
		Scheme:            scheme,
		serviceIdResolver: serviceidresolver.NewResolver(client),
		otterizeClient:    otterizeClient,
	}
}

func (r *NetworkPolicyUploaderReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.NetworkPolicy{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *NetworkPolicyUploaderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logrus.WithField("policy", req.NamespacedName.String()).Debug("Reconcile Otterize NetworkPolicy")

	netpol := &v1.NetworkPolicy{}
	err := r.Get(ctx, req.NamespacedName, netpol)
	if k8serrors.IsNotFound(err) {
		logrus.WithField("policy", req.NamespacedName.String()).Debug("NetPol was deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	mainScopeWorkloads, err := r.getMainScopeWorkloads(ctx, netpol)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	var ingressPolicyRulesAffectedWorkloads []cloudclient.WorkloadIdentityInput
	var egressPolicyRulesAffectedWorkloads []cloudclient.WorkloadIdentityInput

	if lo.Contains(netpol.Spec.PolicyTypes, v1.PolicyTypeIngress) {
		ingressPolicyRulesAffectedWorkloads, err = r.getIngressPolicyRulesAffectedWorkloads(ctx, netpol)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err)
		}
	}

	if lo.Contains(netpol.Spec.PolicyTypes, v1.PolicyTypeEgress) {
		egressPolicyRulesAffectedWorkloads, err = r.getEgressPolicyRulesAffectedWorkloads(ctx, netpol)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err)
		}
	}

	yamlContent, err := yaml.Marshal(netpol)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	err = r.otterizeClient.ReportNetworkPolicy(ctx, cloudclient.NetworkPolicyReportInput{
		Namespace:          nilable.From(netpol.Namespace),
		Name:               netpol.Name,
		MainScopeWorkloads: mainScopeWorkloads,
		IngressWorkloads:   ingressPolicyRulesAffectedWorkloads,
		EgressWorkloads:    egressPolicyRulesAffectedWorkloads,
		Content:            string(yamlContent),
	})
	if err != nil {
		logrus.WithError(err).
			WithField("namespace", req.Namespace).
			Error("failed reporting network policies")
		return ctrl.Result{}, errors.Wrap(err)
	}

	return ctrl.Result{}, nil
}

func (r *NetworkPolicyUploaderReconciler) getMainScopeWorkloads(ctx context.Context, netpol *v1.NetworkPolicy) ([]cloudclient.WorkloadIdentityInput, error) {
	listFilters := []client.ListOption{&client.ListOptions{Namespace: netpol.Namespace}}

	if !isLabelSelectorEmpty(&netpol.Spec.PodSelector) {
		selector, err := metav1.LabelSelectorAsSelector(&netpol.Spec.PodSelector)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		listFilters = append(listFilters, &client.MatchingLabelsSelector{Selector: selector})
	}

	var podList corev1.PodList
	err := r.List(
		ctx, &podList,
		listFilters...)
	if err != nil {
		logrus.WithError(err).Errorf("error when reading podlist")
		return nil, errors.Wrap(err)
	}

	mainScopeWorkloads, err := r.podsToWorkloadIdentities(podList.Items)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return mainScopeWorkloads, nil
}

func (r *NetworkPolicyUploaderReconciler) getIngressPolicyRulesAffectedWorkloads(ctx context.Context, netpol *v1.NetworkPolicy) ([]cloudclient.WorkloadIdentityInput, error) {
	workloads := make([]cloudclient.WorkloadIdentityInput, 0)
	for _, ingressRule := range netpol.Spec.Ingress {
		for _, ingressRuleFrom := range ingressRule.From {
			ingressRuleWorkloads, err := r.getWorkloadsFromNetworkPolicyPeer(ctx, ingressRuleFrom)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			workloads = append(workloads, ingressRuleWorkloads...)
		}
	}
	return workloads, nil
}

func (r *NetworkPolicyUploaderReconciler) getEgressPolicyRulesAffectedWorkloads(ctx context.Context, netpol *v1.NetworkPolicy) ([]cloudclient.WorkloadIdentityInput, error) {
	workloads := make([]cloudclient.WorkloadIdentityInput, 0)
	for _, egressRule := range netpol.Spec.Egress {
		for _, egressRuleTo := range egressRule.To {
			egressRuleWorkloads, err := r.getWorkloadsFromNetworkPolicyPeer(ctx, egressRuleTo)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			workloads = append(workloads, egressRuleWorkloads...)
		}
	}
	return workloads, nil
}

func (r *NetworkPolicyUploaderReconciler) getWorkloadsFromNetworkPolicyPeer(ctx context.Context, peer v1.NetworkPolicyPeer) ([]cloudclient.WorkloadIdentityInput, error) {
	var listFilters []client.ListOption

	if peer.PodSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(peer.PodSelector)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		listFilters = append(listFilters, &client.MatchingLabelsSelector{Selector: selector})
	}

	var namespaces []string
	if peer.NamespaceSelector != nil {
		nsLabelSelector, err := metav1.LabelSelectorAsSelector(peer.NamespaceSelector)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		var namespaceList corev1.NamespaceList
		err = r.List(ctx, &namespaceList, &client.ListOptions{LabelSelector: nsLabelSelector})
		if err != nil {
			return nil, errors.Wrap(err)
		}

		for _, ns := range namespaceList.Items {
			namespaces = append(namespaces, ns.Name)
		}
	}

	if len(namespaces) == 0 {
		var podList corev1.PodList
		err := r.List(ctx, &podList, listFilters...)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		workloads, err := r.podsToWorkloadIdentities(podList.Items)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return workloads, nil
	}

	var allPods []corev1.Pod
	for _, ns := range namespaces {
		var podList corev1.PodList
		err := r.List(ctx, &podList, append(listFilters, client.InNamespace(ns))...)
		if err != nil {
			logrus.WithError(err).Errorf("error when reading podlist for namespace %s", ns)
			return nil, errors.Wrap(err)
		}
		allPods = append(allPods, podList.Items...)
	}

	workloads, err := r.podsToWorkloadIdentities(allPods)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return workloads, nil
}

func isLabelSelectorEmpty(selector *metav1.LabelSelector) bool {
	return selector == nil || (len(selector.MatchLabels) == 00 && len(selector.MatchExpressions) == 0)
}

func (r *NetworkPolicyUploaderReconciler) podsToWorkloadIdentities(pods []corev1.Pod) ([]cloudclient.WorkloadIdentityInput, error) {
	res := make([]cloudclient.WorkloadIdentityInput, 0)
	existingIdentities := goset.NewSet[string]()

	for _, pod := range pods {
		si, err := r.serviceIdResolver.ResolvePodToServiceIdentity(context.Background(), &pod)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if existingIdentities.Contains(si.GetFormattedOtterizeIdentityWithKind()) {
			continue
		}
		existingIdentities.Add(si.GetFormattedOtterizeIdentityWithKind())

		workloadIdentity := cloudclient.WorkloadIdentityInput{
			Namespace: si.Namespace,
			Name:      si.Name,
		}
		if si.Kind != "" {
			workloadIdentity.Kind = nilable.From(si.Kind)
		}
		if si.ResolvedUsingOverrideAnnotation != nil {
			workloadIdentity.ResolvedUsingAnnotation = nilable.From(*si.ResolvedUsingOverrideAnnotation)
		}
		res = append(res, workloadIdentity)
	}

	return res, nil
}
