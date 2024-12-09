package resolvers

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func (r *Resolver) discoverInternalSrcIdentity(ctx context.Context, src *model.RecordedDestinationsForSrc) (model.OtterizeServiceIdentity, error) {
	svc, ok, err := r.kubeFinder.ResolveIPToControlPlane(ctx, src.SrcIP)
	if err != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("could not resolve %s to service: %w", src.SrcIP, err)
	}

	if ok {
		resolutionData := model.IdentityResolutionData{
			Host:        lo.ToPtr(src.SrcIP),
			PodHostname: lo.ToPtr(src.SrcHostname),
		}
		return model.OtterizeServiceIdentity{Name: svc.Name, Namespace: svc.Namespace, KubernetesService: &svc.Name, ResolutionData: &resolutionData}, nil
	}

	srcPod, err := r.kubeFinder.ResolveIPToPod(ctx, src.SrcIP)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) || errors.Is(err, kubefinder.ErrNoPodFound) {
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

	// This function requires "src" to be a pointer.
	// If at some point this function will be called with a non-pointer "src"
	// It may cause a bug because the function will not be able to modify the "src" object of the caller.
	r.filterTargetsAccordingToPodCreationTime(src, srcPod)

	return r.resolveInClusterIdentity(ctx, srcPod)
}

func (r *Resolver) filterTargetsAccordingToPodCreationTime(src *model.RecordedDestinationsForSrc, srcPod *corev1.Pod) {
	filteredDestinations := make([]model.Destination, 0)
	for _, dest := range src.Destinations {
		if srcPod.CreationTimestamp.After(dest.LastSeen) {
			logrus.Debugf("Pod %s was created after capture time %s, ignoring", srcPod.Name, dest.LastSeen)
			continue
		}
		filteredDestinations = append(filteredDestinations, dest)
	}
	src.Destinations = filteredDestinations
}

func (r *Resolver) resolveInClusterIdentity(ctx context.Context, pod *corev1.Pod) (model.OtterizeServiceIdentity, error) {
	if pod.DeletionTimestamp != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("pod %s is being deleted, ignoring", pod.Name)
	}

	svcIdentity, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, pod)
	if err != nil {
		return model.OtterizeServiceIdentity{}, errors.Errorf("could not resolve pod %s to identity: %w", pod.Name, err)
	}

	modelSvcIdentity := model.OtterizeServiceIdentity{
		Name:      svcIdentity.Name,
		Namespace: pod.Namespace,
		Labels:    kubefinder.PodLabelsToOtterizeLabels(pod),
		ResolutionData: &model.IdentityResolutionData{
			Host:        lo.ToPtr(pod.Status.PodIP),
			PodHostname: lo.ToPtr(pod.Name),
			IsService:   lo.ToPtr(false),
			ExtraInfo:   lo.ToPtr("resolveInClusterIdentity"),
		},
	}
	if svcIdentity.OwnerObject != nil {
		modelSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(svcIdentity.OwnerObject.GetObjectKind().GroupVersionKind())
	}

	return modelSvcIdentity, nil
}
