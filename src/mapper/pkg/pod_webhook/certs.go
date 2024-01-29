package pod_webhook

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func copyCA(ctx context.Context, client client.Client, secretName, secretNamespace, targetNamespace string) error {
	exists, err := checkCAExists(ctx, client, secretName, targetNamespace)

	if err != nil {
		return errors.Errorf("unable to check if configmap %s exists in namespace %s: %w", secretName, targetNamespace, err)
	}

	if exists {
		logrus.Debug("CA configmap already exists - no action required")
		return nil
	}

	var secret corev1.Secret
	err = client.Get(ctx, types.NamespacedName{
		Namespace: secretNamespace,
		Name:      secretName,
	}, &secret)

	if err != nil {
		return errors.Errorf("unable to get certificate authority secret %s in namespace %s: %w", secretName, secretNamespace, err)
	}

	certificate := secret.Data["ca.crt"]

	err = writeCA(ctx, client, secretName, targetNamespace, certificate)

	if err != nil {
		return errors.Errorf("unable to write certificate authority to configmap %s in namespace %s: %w", secretName, targetNamespace, err)
	}

	logrus.Infof("copied CA to configmap %s in namespace %s", secretName, targetNamespace)
	return nil
}

func checkCAExists(ctx context.Context, client client.Client, configMapName, configMapNamespace string) (bool, error) {
	var configMap corev1.ConfigMap
	err := client.Get(ctx, types.NamespacedName{
		Namespace: configMapNamespace,
		Name:      configMapName,
	}, &configMap)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func writeCA(ctx context.Context, client client.Client, configMapName, configMapNamespace string, certificate []byte) error {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		BinaryData: map[string][]byte{
			"ca.crt": certificate,
		},
	}

	err := client.Create(ctx, &configMap)

	if err != nil {
		return errors.Errorf("unable to create configmap: %w", err)
	}

	return nil
}
