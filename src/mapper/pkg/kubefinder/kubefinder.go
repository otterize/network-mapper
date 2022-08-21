package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/mapper/pkg/config"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
	"github.com/spf13/viper"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
)

const (
	podIpIndexField = "ip"
)

type KubeFinder struct {
	mgr    manager.Manager
	client client.Client
}

var FoundMoreThanOnePodError = fmt.Errorf("ip belongs to more than one pod")

func NewKubeFinder(mgr manager.Manager) (*KubeFinder, error) {
	indexer := &KubeFinder{client: mgr.GetClient(), mgr: mgr}
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

func (k *KubeFinder) ResolveIpToPod(ctx context.Context, ip string) (*coreV1.Pod, error) {
	var pods coreV1.PodList
	err := k.client.List(ctx, &pods, client.MatchingFields{podIpIndexField: ip})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("pod with ip %s was not found", ip)
	} else if len(pods.Items) > 1 {
		return nil, FoundMoreThanOnePodError
	}
	return &pods.Items[0], nil
}

func (k *KubeFinder) ResolvePodToOtterizeServiceIdentity(ctx context.Context, pod *coreV1.Pod) (model.OtterizeServiceIdentity, error) {
	var obj client.Object
	obj = pod
	for len(obj.GetOwnerReferences()) > 0 {
		owner := obj.GetOwnerReferences()[0]
		ownerObj := &unstructured.Unstructured{}
		ownerObj.SetAPIVersion(owner.APIVersion)
		ownerObj.SetKind(owner.Kind)
		err := k.client.Get(ctx, types.NamespacedName{Name: owner.Name, Namespace: obj.GetNamespace()}, ownerObj)
		if err != nil {
			if errors.IsForbidden(err) {
				// We don't have permissions to the owner object, as we treat it as the identity.
				return model.OtterizeServiceIdentity{Name: owner.Name, Namespace: obj.GetNamespace()}, nil
			}
			return model.OtterizeServiceIdentity{}, err
		}
		obj = ownerObj
	}
	return model.OtterizeServiceIdentity{Name: obj.GetName(), Namespace: obj.GetNamespace()}, nil
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
			There are more forms that records, based on pods hostnames/subdomains/ips, but we ignore them and resolve based on the
			service name for simplicity, as it should be good-enough for intents detection.
		*/
		namespace := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-1]
		serviceName := fqdnWithoutClusterDomainParts[len(fqdnWithoutClusterDomainParts)-2]
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
