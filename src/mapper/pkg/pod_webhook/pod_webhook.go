package pod_webhook

import (
	"context"
	"encoding/json"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type InjectDNSConfigToPodWebhook struct {
	client  client.Client
	decoder *admission.Decoder
}

func NewInjectDNSConfigToPodWebhook(client client.Client, decoder *admission.Decoder) (*InjectDNSConfigToPodWebhook, error) {
	return &InjectDNSConfigToPodWebhook{
		client:  client,
		decoder: decoder,
	}, nil
}

func (w *InjectDNSConfigToPodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := corev1.Pod{}
	err := (*w.decoder).Decode(req, &pod)

	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	logger := logrus.WithField("webhook", "InjectDNSConfigToPodWebhook").
		WithField("pod", pod.Name).
		WithField("namespace", pod.Namespace)
	logger.Debug("Handling Pod")

	if pod.Labels == nil {
		return admission.Allowed("no AWS visibility label - no modifications made")
	}

	_, labelExists := pod.Labels["network-mapper.otterize.com/aws-visibility"]

	if !labelExists {
		logger.Debug("pod doesn't have AWS visibility label, skipping")
		return admission.Allowed("no AWS visibility label - no modifications made")
	}

	currentNamespace, err := kubeutils.GetCurrentNamespace()

	if err != nil {
		logger.WithError(err).Error("unable to get pod namespace")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	dnsServerAddress, err := getDNSServerAddress(w.client, currentNamespace)

	if err != nil {
		logger.WithError(err).Error("unable to get DNS server address")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	err = copyCA(
		ctx,
		w.client,
		"iamlive-ca",
		currentNamespace,
		pod.Namespace,
	)

	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "iamlive-ca",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "iamlive-ca",
				},
			},
		},
	})

	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      "iamlive-ca",
			MountPath: "/tmp/otterize.com/certificates",
			ReadOnly:  true,
		})

		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
			Name:  "AWS_CA_BUNDLE",
			Value: "/tmp/otterize.com/certificates/ca.crt",
		})
	}

	pod.Spec.DNSPolicy = corev1.DNSNone
	pod.Spec.DNSConfig = &corev1.PodDNSConfig{
		Nameservers: []string{dnsServerAddress},
		Searches: []string{
			"default.svc.cluster.local",
			"svc.cluster.local",
			"cluster.local",
			"ec2.internal",
		},
	}

	marshaledPod, err := json.Marshal(pod)

	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
