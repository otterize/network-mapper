package pod_webhook

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getDNSServerAddress(client client.Client, podNamespace string) (string, error) {
	var service corev1.Service
	err := client.Get(context.Background(), types.NamespacedName{
		Namespace: podNamespace,
		Name:      "otterize-dns",
	}, &service)

	if err != nil {
		return "", errors.Errorf("unable to get otterize-dns service: %w", err)
	}

	logrus.Infof("otterize-dns service address: %s", service.Spec.ClusterIP)
	return service.Spec.ClusterIP, nil
}
