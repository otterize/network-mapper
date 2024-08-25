package resolvers

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/sirupsen/logrus"
	"inet.af/netaddr"
	corev1 "k8s.io/api/core/v1"
	"sync"
)

var ipset *netaddr.IPSet
var ipsetOnce sync.Once

func MustGetInternalIPSet() *netaddr.IPSet {
	ipsetOnce.Do(func() {
		setBuilder := netaddr.IPSetBuilder{}
		setBuilder.AddPrefix(netaddr.IPPrefixFrom(netaddr.IPFrom4([4]byte{192, 168, 0, 0}), 16))
		setBuilder.AddPrefix(netaddr.IPPrefixFrom(netaddr.IPFrom4([4]byte{10, 0, 0, 0}), 8))
		setBuilder.AddPrefix(netaddr.IPPrefixFrom(netaddr.IPFrom4([4]byte{172, 16, 0, 0}), 12))
		set, err := setBuilder.IPSet()
		if err != nil {
			logrus.WithError(err).Panic("could not create IP set")
		}
		ipset = set
	})
	return ipset
}

// isExternalIP returns true if the IP is external.
func (r *Resolver) isExternalIP(ctx context.Context, srcIP string) (bool, error) {
	_, err := r.kubeFinder.ResolveIPToPod(ctx, srcIP)
	if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
		return false, nil
	}

	// If the IP is not found, it may be external
	if err != nil && !errors.Is(err, kubefinder.ErrNoPodFound) {
		return false, errors.Wrap(err)
	}

	if MustGetInternalIPSet().Contains(netaddr.MustParseIP(srcIP)) {
		return false, nil
	}

	return true, nil
}

func (r *Resolver) discoverSrcIdentity(ctx context.Context, src model.RecordedDestinationsForSrc) (model.OtterizeServiceIdentity, error) {
	svc, ok, err := r.kubeFinder.ResolveIPToControlPlane(ctx, src.SrcIP)
	if err != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("could not resolve %s to service: %w", src.SrcIP, err)
	}

	if ok {
		return model.OtterizeServiceIdentity{Name: svc.Name, Namespace: svc.Namespace, KubernetesService: &svc.Name}, nil
	}

	srcPod, err := r.kubeFinder.ResolveIPToPod(ctx, src.SrcIP)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			return model.OtterizeServiceIdentity{}, errors.Errorf("IP %s belongs to more than one pod, ignoring", src.SrcIP)
		}
		if errors.Is(err, kubefinder.ErrNoPodFound) {
			return model.OtterizeServiceIdentity{}, errors.Wrap(err)
		}
		return model.OtterizeServiceIdentity{}, errors.Errorf("could not resolve %s to pod: %w", src.SrcIP, err)
	}
	// When running on AWS - we must validate the hostname because the IP may be reused by a new pod (AWS VPC CNI)
	// When not running on AWS - source hostname resolution in the sniffer might be disabled
	if (src.SrcHostname != "" || r.isRunningOnAws) && srcPod.Name != src.SrcHostname {
		// This could mean a new pod is reusing the same IP
		// TODO: Use the captured hostname to actually find the relevant pod (instead of the IP that might no longer exist or be reused)
		return model.OtterizeServiceIdentity{}, errors.Errorf("found pod %s (by ip %s) doesn't match captured hostname %s, ignoring", srcPod.Name, src.SrcIP, src.SrcHostname)
	}

	filteredDestinations := make([]model.Destination, 0)
	for _, dest := range src.Destinations {
		if srcPod.CreationTimestamp.After(dest.LastSeen) {
			logrus.Debugf("Pod %s was created after capture time %s, ignoring", srcPod.Name, dest.LastSeen)
			continue
		}
		filteredDestinations = append(filteredDestinations, dest)
	}
	src.Destinations = filteredDestinations

	return r.resolveInClusterIdentity(ctx, srcPod)
}

func (r *Resolver) resolveInClusterIdentity(ctx context.Context, pod *corev1.Pod) (model.OtterizeServiceIdentity, error) {
	if pod.DeletionTimestamp != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("pod %s is being deleted, ignoring", pod.Name)
	}

	svcIdentity, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, pod)
	if err != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("could not resolve pod %s to identity: %w", pod.Name, err)
	}

	modelSvcIdentity := model.OtterizeServiceIdentity{Name: svcIdentity.Name, Namespace: pod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(pod)}
	if svcIdentity.OwnerObject != nil {
		modelSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(svcIdentity.OwnerObject.GetObjectKind().GroupVersionKind())
	}

	return modelSvcIdentity, nil
}
