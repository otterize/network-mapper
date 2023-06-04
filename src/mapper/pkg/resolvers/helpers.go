package resolvers

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/operator/api/v1alpha2"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
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

func (r *mutationResolver) getServiceLabels(ctx context.Context, service *model.ResolvedOtterizeServiceIdentityInput) []model.PodLabel {
	intent := v1alpha2.Intent{
		Name: fmt.Sprintf("%s.%s", service.Name, service.Namespace),
	}

	pod, err := r.serviceIdResolver.ResolveIntentServerToPod(ctx, intent, service.Namespace)
	if err != nil {
		logrus.WithError(err).Warningf("Error getting labels for service %s.%s", service.Name, service.Namespace)
		return nil
	}

	return podLabelsToOtterizeLabels(&pod)
}
