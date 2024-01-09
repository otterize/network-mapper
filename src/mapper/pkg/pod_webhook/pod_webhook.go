package pod_webhook

import (
	"context"
	"encoding/json"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type InjectDNSConfigToPodWebhook struct {
	client           client.Client
	decoder          *admission.Decoder
	dnsServerAddress string
}

func NewInjectDNSConfigToPodWebhook(client client.Client, decoder *admission.Decoder) *InjectDNSConfigToPodWebhook {
	podNamespace, err := kubeutils.GetCurrentNamespace()

	if err != nil {
		logrus.WithError(err).Panic("unable to get pod namespace")
	}

	var service corev1.Service
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: podNamespace,
		Name:      "otterize-dns",
	}, &service)

	if err != nil {
		logrus.WithError(err).Panic("unable to get otterize-dns service address")
	}

	logrus.Infof("otterize-dns service address: %s", service.Spec.ClusterIP)

	return &InjectDNSConfigToPodWebhook{
		client:           client,
		decoder:          decoder,
		dnsServerAddress: service.Spec.ClusterIP,
	}
}

func (w *InjectDNSConfigToPodWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := corev1.Pod{}
	err := w.decoder.Decode(req, &pod)

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

	pod.Spec.DNSPolicy = corev1.DNSNone
	pod.Spec.DNSConfig = &corev1.PodDNSConfig{
		Nameservers: []string{w.dnsServerAddress},
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
