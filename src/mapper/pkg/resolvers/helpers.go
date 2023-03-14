package resolvers

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
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
