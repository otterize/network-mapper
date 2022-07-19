package kubefinder

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/mapper/pkg/config"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
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

func (k *KubeFinder) ResolvePodToServiceIdentity(ctx context.Context, pod *coreV1.Pod) (model.ServiceIdentity, error) {
	for _, owner := range pod.OwnerReferences {
		namespacedName := types.NamespacedName{Name: owner.Name, Namespace: pod.Namespace}
		switch owner.Kind {
		case "ReplicaSet":
			rs := &appsV1.ReplicaSet{}
			err := k.client.Get(ctx, namespacedName, rs)
			if err != nil {
				return model.ServiceIdentity{}, err
			}
			return model.ServiceIdentity{Name: rs.OwnerReferences[0].Name, Namespace: pod.Namespace}, nil
		case "DaemonSet":
			ds := &appsV1.DaemonSet{}
			err := k.client.Get(ctx, namespacedName, ds)
			if err != nil {
				return model.ServiceIdentity{}, err
			}
			return model.ServiceIdentity{Name: ds.Name, Namespace: pod.Namespace}, nil
		case "StatefulSet":
			ss := &appsV1.StatefulSet{}
			err := k.client.Get(ctx, namespacedName, ss)
			if err != nil {
				return model.ServiceIdentity{}, err
			}
			return model.ServiceIdentity{Name: ss.Name, Namespace: pod.Namespace}, nil
		default:
			logrus.Infof("Unknown owner kind %s for pod %s", owner.Kind, pod.Name)
		}
	}
	return model.ServiceIdentity{}, fmt.Errorf("pod %s has no owner", pod.Name)
}

func (k *KubeFinder) ResolveServiceAddressToIps(ctx context.Context, fqdn string) ([]string, error) {
	if !strings.HasSuffix(fqdn, viper.GetString(config.ClusterDomainKey)) {
		return nil, fmt.Errorf("address %s is not in the cluster", fqdn)
	}
	endpointName := strings.Split(fqdn, ".")[0]
	namespace := strings.Split(fqdn, ".")[1]
	endpoint := &coreV1.Endpoints{}
	err := k.client.Get(ctx, types.NamespacedName{Name: endpointName, Namespace: namespace}, endpoint)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0)
	for _, subset := range endpoint.Subsets {
		for _, address := range subset.Addresses {
			ips = append(ips, address.IP)
		}
	}
	return ips, nil
}
