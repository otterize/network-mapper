package resolvers

import (
	"context"
	"errors"
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

func (r *mutationResolver) discoverSrcIdentity(ctx context.Context, src model.RecordedDestinationsForSrc) *model.OtterizeServiceIdentity {
	srcPod, err := r.kubeFinder.ResolveIPToPod(ctx, src.SrcIP)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", src.SrcIP)
		} else {
			logrus.WithError(err).Debugf("Could not resolve %s to pod", src.SrcIP)
		}
		return nil
	}
	if src.SrcHostname != "" && srcPod.Name != src.SrcHostname {
		// This could mean a new pod is reusing the same IP
		// TODO: Use the captured hostname to actually find the relevant pod (instead of the IP that might no longer exist or be reused)
		logrus.Warnf("Found pod %s (by ip %s) doesn't match captured hostname %s, ignoring", srcPod.Name, src.SrcIP, src.SrcHostname)
		return nil
	}

	if srcPod.DeletionTimestamp != nil {
		logrus.Debugf("Pod %s is being deleted, ignoring", srcPod.Name)
		return nil
	}

	srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
	if err != nil {
		logrus.WithError(err).Debugf("Could not resolve pod %s to identity", srcPod.Name)
		return nil
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

	return &srcSvcIdentity
}
