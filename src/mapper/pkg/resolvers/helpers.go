package resolvers

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/sirupsen/logrus"
)

// isExternalOrAssumeExternalIfError returns true if the IP is external or if an error occurred while determining if the IP is external.
func (r *Resolver) isExternalOrAssumeExternalIfError(ctx context.Context, srcIP string) (bool, error) {
	_, err := r.kubeFinder.ResolveIPToPod(ctx, srcIP)
	if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
		return false, nil
	}

	// If the IP is not found, it may be external
	if err != nil && !errors.Is(err, kubefinder.ErrNoPodFound) {
		return false, errors.Wrap(err)
	}

	// if the IP is not in any node's pod CIDR, it is external.
	isExternal, err := r.kubeFinder.IsIPNotInNodePodCIDR(ctx, srcIP)
	if err != nil {
		logrus.WithError(err).WithField("ip", srcIP).Debug("could not determine if IP is external, assuming it is")
		return true, nil
	}
	return isExternal, nil
}

func (r *Resolver) discoverSrcIdentity(ctx context.Context, src model.RecordedDestinationsForSrc) (model.OtterizeServiceIdentity, error) {
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

	if srcPod.DeletionTimestamp != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("pod %s is being deleted, ignoring", srcPod.Name)
	}

	srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
	if err != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("could not resolve pod %s to identity: %w", srcPod.Name, err)
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
	srcSvcIdentity := model.OtterizeServiceIdentity{Name: srcService.Name, Namespace: srcPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(srcPod)}
	if srcService.OwnerObject != nil {
		srcSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(srcService.OwnerObject.GetObjectKind().GroupVersionKind())
	}

	return srcSvcIdentity, nil
}
