package kubefinder

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"inet.af/netaddr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
)

const (
	podIPIndexField            = "ip"
	serviceIPIndexField        = "spec.ip"
	externalIPIndexField       = "spec.externalIPs"
	IstioCanonicalNameLabelKey = "service.istio.io/canonical-name"
)

type KubeFinder struct {
	mgr               manager.Manager
	client            client.Client
	serviceIdResolver *serviceidresolver.Resolver
}

var ErrNoPodFound = errors.Errorf("no pod found")
var ErrFoundMoreThanOnePod = errors.Errorf("ip belongs to more than one pod")
var ErrFoundMoreThanOneService = errors.Errorf("ip belongs to more than one service")

func NewKubeFinder(ctx context.Context, mgr manager.Manager) (*KubeFinder, error) {
	indexer := &KubeFinder{client: mgr.GetClient(), mgr: mgr, serviceIdResolver: serviceidresolver.NewResolver(mgr.GetClient())}
	err := indexer.initIndexes(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return indexer, nil
}

func (k *KubeFinder) initIndexes(ctx context.Context) error {
	err := k.mgr.GetCache().IndexField(ctx, &corev1.Pod{}, podIPIndexField, func(object client.Object) []string {
		res := make([]string, 0)
		pod := object.(*corev1.Pod)
		for _, ip := range pod.Status.PodIPs {
			if pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
				continue
			}
			res = append(res, ip.IP)
		}
		return res
	})
	if err != nil {
		return errors.Wrap(err)
	}

	err = k.mgr.GetCache().IndexField(ctx, &corev1.Service{}, serviceIPIndexField, func(object client.Object) []string {
		res := make([]string, 0)
		svc := object.(*corev1.Service)
		res = append(res, svc.Spec.ClusterIPs...)
		return res
	})
	if err != nil {
		return errors.Wrap(err)
	}

	err = k.mgr.GetCache().IndexField(ctx, &corev1.Service{}, externalIPIndexField, func(object client.Object) []string {
		ips := sets.New[string]()
		svc := object.(*corev1.Service)
		if svc.DeletionTimestamp != nil {
			return nil
		}
		if svc.Spec.Type == corev1.ServiceTypeNodePort {
			return nil
		}

		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			ips.Insert(ingress.IP)
		}
		return ips.UnsortedList()
	})
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (k *KubeFinder) ResolvePodByName(ctx context.Context, name string, namespace string) (*corev1.Pod, error) {
	var pod corev1.Pod
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pod)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &pod, nil
}

func (k *KubeFinder) ResolveIPToService(ctx context.Context, ip string) (*corev1.Service, bool, error) {
	var services corev1.ServiceList
	err := k.client.List(ctx, &services, client.MatchingFields{serviceIPIndexField: ip})
	if err != nil {
		return nil, false, errors.Wrap(err)
	}
	if len(services.Items) == 0 {
		return nil, false, nil
	}

	if len(services.Items) != 1 {
		return nil, false, ErrFoundMoreThanOneService
	}
	return &services.Items[0], true, nil
}

func (k *KubeFinder) ResolveServiceToPods(ctx context.Context, svc *corev1.Service) ([]corev1.Pod, error) {
	var endpoints corev1.Endpoints
	err := k.client.Get(ctx, types.NamespacedName{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	}, &endpoints)
	if err != nil {
		return nil, errors.Wrap(err)
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
			if k8serrors.IsNotFound(err) {
				continue
			}
			return nil, errors.Wrap(err)
		}
		if pod.DeletionTimestamp != nil {
			continue
		}
		pods = append(pods, pod)
	}

	return pods, nil
}

func (k *KubeFinder) IsExternalIP(ctx context.Context, ip string) (bool, error) {
	var nodes corev1.NodeList
	err := k.client.List(ctx, &nodes)
	if err != nil {
		return false, errors.Wrap(err)
	}

	cidrBuilder := netaddr.IPSetBuilder{}
	for _, node := range nodes.Items {
		nodeCidr := node.Spec.PodCIDR
		if nodeCidr == "" {
			logrus.Errorf("node %s has no podCIDR", node.Name)
			continue
		}

		logrus.Debugf("node %s has podCIDR %s", node.Name, nodeCidr)
		cidr, err := netaddr.ParseIPPrefix(nodeCidr)
		if err != nil {
			return false, errors.Wrap(err)
		}

		cidrBuilder.AddPrefix(cidr)
	}

	cidrSet, err := cidrBuilder.IPSet()
	if err != nil {
		return false, errors.Wrap(err)
	}

	ipAddr, err := netaddr.ParseIP(ip)
	if err != nil {
		return false, errors.Wrap(err)
	}

	return !cidrSet.Contains(ipAddr), nil
}

func (k *KubeFinder) ResolveIPToPod(ctx context.Context, ip string) (*corev1.Pod, error) {
	var pods corev1.PodList
	err := k.client.List(ctx, &pods, client.MatchingFields{podIPIndexField: ip})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(pods.Items) == 0 {
		return nil, errors.Wrap(ErrNoPodFound)
	}

	if len(pods.Items) != 1 {
		return nil, errors.Wrap(ErrFoundMoreThanOnePod)
	}
	return &pods.Items[0], nil
}

func (k *KubeFinder) ResolveIPToExternalAccessService(ctx context.Context, ip string, port int) (*corev1.Service, bool, error) {
	var services corev1.ServiceList
	err := k.client.List(ctx, &services, client.MatchingFields{externalIPIndexField: ip})
	if err != nil {
		return nil, false, errors.Wrap(err)
	}
	if len(services.Items) == 0 {
		return nil, false, nil
	}
	if len(services.Items) != 1 {
		return nil, false, errors.Wrap(ErrFoundMoreThanOneService)
	}
	service := services.Items[0]
	_, isServicePort := lo.Find(service.Spec.Ports, func(p corev1.ServicePort) bool {
		return p.Port == int32(port)
	})

	if !isServicePort {
		logrus.Debugf("service %s does not have port %d, ignoring", service.Name, port)
		return nil, false, nil
	}

	return &services.Items[0], true, nil
}

func (k *KubeFinder) ResolveIstioWorkloadToPod(ctx context.Context, workload string, namespace string) (*corev1.Pod, error) {
	podList := corev1.PodList{}
	err := k.client.List(ctx, &podList, client.InNamespace(namespace), client.MatchingLabels{IstioCanonicalNameLabelKey: workload})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	// Cannot happen theoretically
	if len(podList.Items) == 0 {
		return nil, errors.Errorf("no matching pods for workload %s", workload)
	}

	return &podList.Items[0], nil
}

func (k *KubeFinder) ResolveServiceAddressToIps(ctx context.Context, fqdn string) ([]string, types.NamespacedName, error) {
	clusterDomain := viper.GetString(config.ClusterDomainKey)
	if !strings.HasSuffix(fqdn, clusterDomain) {
		return nil, types.NamespacedName{}, errors.Errorf("address %s is not in the cluster", fqdn)
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
			return nil, types.NamespacedName{}, errors.Errorf("service address %s is too short", fqdn)
		}
		namespace := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-2]
		serviceName := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-3]
		endpoints := &corev1.Endpoints{}
		serviceNamespacedName := types.NamespacedName{Name: serviceName, Namespace: namespace}
		err := k.client.Get(ctx, serviceNamespacedName, endpoints)
		if err != nil {
			return nil, types.NamespacedName{}, errors.Wrap(err)
		}
		ips := make([]string, 0)
		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				ips = append(ips, address.IP)
			}
		}
		return ips, serviceNamespacedName, nil
	case "pod":
		// for address format of pods: 172-17-0-3.default.pod.cluster.local
		return []string{strings.ReplaceAll(fqdnWithoutClusterDomainParts[0], "-", ".")}, types.NamespacedName{}, nil
	default:
		return nil, types.NamespacedName{}, errors.Errorf("cannot resolve k8s address %s, type %s not supported", fqdn, fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-1])
	}
}
