package ipresolver

import (
	"context"
	"errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/shared/kubefinder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"time"
)

var NotK8sService = errors.New("not a k8s service")
var NotPodAddress = errors.New("not a pod address")

type Identity struct {
	Name      string
	Namespace string
}

type PodIP string
type DestDNS string

type PodResolver interface {
	ResolveIP(ip PodIP, captureTime time.Time) (Identity, error)
	ResolveDNS(dns DestDNS, captureTime time.Time) (Identity, error)
	WaitForUpdateTime(ctx context.Context, updateTime time.Time) error
}

type PodResolverImpl struct {
	serviceResolver *serviceidresolver.Resolver
	finder          *kubefinder.KubeFinder
}

func NewPodResolver(finder *kubefinder.KubeFinder, serviceResolver *serviceidresolver.Resolver) *PodResolverImpl {
	return &PodResolverImpl{
		serviceResolver: serviceResolver,
		finder:          finder,
	}
}

func (r *PodResolverImpl) WaitForUpdateTime(ctx context.Context, updateTime time.Time) error {
	return r.finder.WaitForUpdateTime(ctx, updateTime)
}

func (r *PodResolverImpl) ResolveIP(podIP PodIP, captureTime time.Time) (Identity, error) {
	ctx := context.Background()
	pod, err := r.finder.ResolveIpToPod(ctx, string(podIP))
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", podIP)
		} else {
			logrus.WithError(err).Debugf("Could not resolve %s to pod", podIP)
		}
		return Identity{}, err
	}

	if pod.Status.StartTime.After(captureTime) {
		return Identity{}, errors.New("pod was created after the capture time, can't resolve IP owner")
	}

	service, err := r.serviceResolver.ResolvePodToServiceIdentity(ctx, pod)
	if err != nil {
		logrus.WithError(err).Debugf("Could not resolve pod %s to identity", pod.Name)
		return Identity{}, err
	}

	otterizeIdentity := Identity{
		Name:      service.Name,
		Namespace: pod.Namespace,
	}

	return otterizeIdentity, nil
}

func (r *PodResolverImpl) ResolveDNS(dns DestDNS, captureTime time.Time) (Identity, error) {
	if !strings.HasSuffix(string(dns), viper.GetString(config.ClusterDomainKey)) {
		logrus.Debugf("DNS %s does not belong to cluster domain, ignoring", dns)
		return Identity{}, NotK8sService
	}

	ips, err := r.finder.ResolveServiceAddressToIps(context.Background(), string(dns))
	if err != nil {
		return Identity{}, err
	}

	if len(ips) == 0 {
		logrus.Debugf("Service address %s is currently not backed by any pod, ignoring", dns)
		return Identity{}, NotPodAddress
	}

	// TODO: Explain why we are using the first IP
	serviceName, err := r.ResolveIP(PodIP(ips[0]), captureTime)
	if err != nil {
		return Identity{}, err
	}

	return serviceName, nil
}

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
