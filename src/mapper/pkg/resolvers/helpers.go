package resolvers

import (
	"context"
	"errors"
	"fmt"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func podLabelsToOtterizeLabels(pod *corev1.Pod) []model.PodLabel {
	labels := make([]model.PodLabel, 0, len(pod.Labels))
	for key, value := range pod.Labels {
		labels = append(labels, model.PodLabel{
			Key:   key,
			Value: value,
		})
	}

	return labels
}

func serviceLabelsToOtterizeLabels(service *corev1.Service) []model.PodLabel {
	labels := make([]model.PodLabel, 0)
	for key, value := range service.Labels {
		labels = append(labels, model.PodLabel{
			Key:   key,
			Value: value,
		})
	}

	return labels
}

func (r *Resolver) discoverSrcIdentity(ctx context.Context, src model.RecordedDestinationsForSrc) (model.OtterizeServiceIdentity, error) {
	srcPod, err := r.kubeFinder.ResolveIPToPod(ctx, src.SrcIP)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			return model.OtterizeServiceIdentity{}, fmt.Errorf("IP %s belongs to more than one pod, ignoring", src.SrcIP)
		}
		return model.OtterizeServiceIdentity{}, fmt.Errorf("could not resolve %s to pod: %w", src.SrcIP, err)
	}
	if src.SrcHostname != "" && srcPod.Name != src.SrcHostname {
		// This could mean a new pod is reusing the same IP
		// TODO: Use the captured hostname to actually find the relevant pod (instead of the IP that might no longer exist or be reused)
		return model.OtterizeServiceIdentity{}, fmt.Errorf("found pod %s (by ip %s) doesn't match captured hostname %s, ignoring", srcPod.Name, src.SrcIP, src.SrcHostname)
	}

	if srcPod.DeletionTimestamp != nil {
		return model.OtterizeServiceIdentity{}, fmt.Errorf("pod %s is being deleted, ignoring", srcPod.Name)
	}

	srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
	if err != nil {
		return model.OtterizeServiceIdentity{}, fmt.Errorf("could not resolve pod %s to identity: %w", srcPod.Name, err)
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
	srcSvcIdentity := model.OtterizeServiceIdentity{Name: srcService.Name, Namespace: srcPod.Namespace, Labels: podLabelsToOtterizeLabels(srcPod)}
	if srcService.OwnerObject != nil {
		srcSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(srcService.OwnerObject.GetObjectKind().GroupVersionKind())
	}

	return srcSvcIdentity, nil
}
