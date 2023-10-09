package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
)

const (
	podIPIndexField            = "ip"
	serviceIPIndexField        = "spec.ip"
	IstioCanonicalNameLabelKey = "service.istio.io/canonical-name"
)

type KubeFinder struct {
	mgr               manager.Manager
	client            client.Client
	serviceIdResolver *serviceidresolver.Resolver
}

var ErrFoundMoreThanOnePod = fmt.Errorf("ip belongs to more than one pod")
var ErrFoundMoreThanOneService = fmt.Errorf("ip belongs to more than one service")

func NewKubeFinder(ctx context.Context, mgr manager.Manager) (*KubeFinder, error) {
	indexer := &KubeFinder{client: mgr.GetClient(), mgr: mgr, serviceIdResolver: serviceidresolver.NewResolver(mgr.GetClient())}
	err := indexer.initIndexes(ctx)
	if err != nil {
		return nil, err
	}
	return indexer, nil
}

func (k *KubeFinder) initIndexes(ctx context.Context) error {
	err := k.mgr.GetCache().IndexField(ctx, &corev1.Pod{}, podIPIndexField, func(object client.Object) []string {
		res := make([]string, 0)
		pod := object.(*corev1.Pod)
		for _, ip := range pod.Status.PodIPs {
			res = append(res, ip.IP)
		}
		return res
	})
	if err != nil {
		return err
	}

	err = k.mgr.GetCache().IndexField(ctx, &corev1.Service{}, serviceIPIndexField, func(object client.Object) []string {
		res := make([]string, 0)
		svc := object.(*corev1.Service)
		res = append(res, svc.Spec.ClusterIPs...)
		return res
	})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubeFinder) ResolvePodByName(ctx context.Context, name string, namespace string) (*corev1.Pod, error) {
	var pod corev1.Pod
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pod)
	if err != nil {
		return nil, err
	}

	return &pod, nil
}

func (k *KubeFinder) ResolveIPToService(ctx context.Context, ip string) (*corev1.Service, error) {
	var services corev1.ServiceList
	err := k.client.List(ctx, &services, client.MatchingFields{serviceIPIndexField: ip})
	if err != nil {
		return nil, err
	}
	if len(services.Items) == 0 {
		return nil, errors.NewNotFound(corev1.Resource("service"), ip)
	}

	if len(services.Items) != 1 {
		return nil, ErrFoundMoreThanOneService
	}
	return &services.Items[0], nil
}

func (k *KubeFinder) ResolveServiceToPods(ctx context.Context, svc *corev1.Service) ([]corev1.Pod, error) {
	var endpoints corev1.Endpoints
	err := k.client.Get(ctx, types.NamespacedName{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	}, &endpoints)
	if err != nil {
		return nil, err
	}

	pods := make([]corev1.Pod, 0)

	addresses := make([]corev1.EndpointAddress, 0)
	for _, subset := range endpoints.Subsets {
		addresses = append(addresses, subset.Addresses...)
		addresses = append(addresses, subset.NotReadyAddresses...)
	}

	for _, address := range addresses {
		if address.TargetRef == nil || address.TargetRef.Kind != "Pod" {
			continue
		}
		var pod corev1.Pod
		err := k.client.Get(ctx, types.NamespacedName{Name: address.TargetRef.Name, Namespace: address.TargetRef.Namespace}, &pod)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		if pod.DeletionTimestamp != nil {
			continue
		}
		pods = append(pods, pod)
	}

	return pods, nil
}

func (k *KubeFinder) ResolveIPToPod(ctx context.Context, ip string) (*corev1.Pod, error) {
	var pods corev1.PodList
	err := k.client.List(ctx, &pods, client.MatchingFields{podIPIndexField: ip})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, errors.NewNotFound(corev1.Resource("pod"), ip)
	}

	if len(pods.Items) != 1 {
		return nil, ErrFoundMoreThanOnePod
	}
	return &pods.Items[0], nil
}

func (k *KubeFinder) ResolveIstioWorkloadToPod(ctx context.Context, workload string, namespace string) (*corev1.Pod, error) {
	podList := corev1.PodList{}
	err := k.client.List(ctx, &podList, client.InNamespace(namespace), client.MatchingLabels{IstioCanonicalNameLabelKey: workload})
	if err != nil {
		return nil, err
	}
	// Cannot happen theoretically
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no matching pods for workload %s", workload)
	}

	return &podList.Items[0], nil
}

func (k *KubeFinder) ResolveServiceAddressToIps(ctx context.Context, fqdn string) ([]string, string, error) {
	clusterDomain := viper.GetString(config.ClusterDomainKey)
	if !strings.HasSuffix(fqdn, clusterDomain) {
		return nil, "", fmt.Errorf("address %s is not in the cluster", fqdn)
	}
	fqdnWithoutClusterDomain := fqdn[:len(fqdn)-len("."+clusterDomain)]
	fqdnWithoutClusterDomainParts := strings.Split(fqdnWithoutClusterDomain, ".")
	switch fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-1] {
	case "svc":
		/*
			The basic form of service record is service-name.my-namespace.svc.cluster-domain.example
			There are more forms of records, based on pods hostnames/subdomains/ips, but we ignore them and resolve based on the
			service name for simplicity, as it should be good enough for intents detection.
		*/
		if len(fqdnWithoutClusterDomainParts) < 3 {
			// expected at least service-name.namespace.svc
			return nil, "", fmt.Errorf("service address %s is too short", fqdn)
		}
		namespace := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-2]
		serviceName := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-3]
		endpoints := &corev1.Endpoints{}
		err := k.client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, endpoints)
		if err != nil {
			return nil, "", err
		}
		ips := make([]string, 0)
		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				ips = append(ips, address.IP)
			}
		}
		return ips, serviceName, nil
	case "pod":
		// for address format of pods: 172-17-0-3.default.pod.cluster.local
		return []string{strings.ReplaceAll(fqdnWithoutClusterDomainParts[0], "-", ".")}, "", nil
	default:
		return nil, "", fmt.Errorf("cannot resolve k8s address %s, type %s not supported", fqdn, fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-1])
	}
}
