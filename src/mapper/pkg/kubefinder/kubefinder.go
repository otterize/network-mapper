package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
)

const (
	podIpIndexField            = "ip"
	IstioCanonicalNameLabelKey = "service.istio.io/canonical-name"
)

type KubeFinder struct {
	mgr               manager.Manager
	client            client.Client
	serviceIdResolver *serviceidresolver.Resolver
}

var ErrFoundMoreThanOnePod = fmt.Errorf("ip belongs to more than one pod")

func NewKubeFinder(mgr manager.Manager) (*KubeFinder, error) {
	indexer := &KubeFinder{client: mgr.GetClient(), mgr: mgr, serviceIdResolver: serviceidresolver.NewResolver(mgr.GetClient())}
	err := indexer.initIndexes()
	if err != nil {
		return nil, err
	}
	return indexer, nil
}

func (k *KubeFinder) initIndexes() error {
	err := k.mgr.GetCache().IndexField(context.Background(), &coreV1.Pod{}, podIpIndexField, func(object client.Object) []string {
		res := make([]string, 0)
		pod := object.(*coreV1.Pod)
		for _, ip := range pod.Status.PodIPs {
			res = append(res, ip.IP)
		}
		return res
	})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubeFinder) ResolvePodByName(ctx context.Context, name string, namespace string) (*coreV1.Pod, error) {
	var pod coreV1.Pod
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pod)
	if err != nil {
		return nil, err
	}

	return &pod, nil
}

func (k *KubeFinder) ResolveIpToPod(ctx context.Context, ip string) (*coreV1.Pod, error) {
	var pods coreV1.PodList
	err := k.client.List(ctx, &pods, client.MatchingFields{podIpIndexField: ip})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("pod with ip %s was not found", ip)
	} else if len(pods.Items) > 1 {
		return nil, ErrFoundMoreThanOnePod
	}
	return &pods.Items[0], nil
}

func (k *KubeFinder) ResolveIstioWorkloadToPod(ctx context.Context, workload string, namespace string) (*coreV1.Pod, error) {
	podList := coreV1.PodList{}
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

func (k *KubeFinder) ResolveServiceAddressToIps(ctx context.Context, fqdn string) ([]string, error) {
	clusterDomain := viper.GetString(config.ClusterDomainKey)
	if !strings.HasSuffix(fqdn, clusterDomain) {
		return nil, fmt.Errorf("address %s is not in the cluster", fqdn)
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
			return nil, fmt.Errorf("service address %s is too short", fqdn)
		}
		namespace := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-2]
		serviceName := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-3]
		endpoints := &coreV1.Endpoints{}
		err := k.client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, endpoints)
		if err != nil {
			return nil, err
		}
		ips := make([]string, 0)
		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				ips = append(ips, address.IP)
			}
		}
		return ips, nil
	case "pod":
		// for address format of pods: 172-17-0-3.default.pod.cluster.local
		return []string{strings.ReplaceAll(fqdnWithoutClusterDomainParts[0], "-", ".")}, nil
	default:
		return nil, fmt.Errorf("cannot resolve k8s address %s, type %s not supported", fqdn, fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-1])
	}
}
