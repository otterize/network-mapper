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

type IpResolver interface {
	ResolveIp(ip string, captureTime time.Time) (Identity, error)
	ResolveDNS(dns string, captureTime time.Time) (Identity, error)
}

type ipResolverImpl struct {
	serviceResolver *serviceidresolver.Resolver
	finder          kubefinder.KubeFinder
}

func NewIpResolver(finder kubefinder.KubeFinder, serviceResolver *serviceidresolver.Resolver) IpResolver {
	return &ipResolverImpl{
		serviceResolver: serviceResolver,
		finder:          finder,
	}
}

func (r *ipResolverImpl) ResolveIp(podIP string, captureTime time.Time) (Identity, error) {
	ctx := context.Background()
	pod, err := r.finder.ResolveIpToPod(ctx, podIP)
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

	startTime := time.Now()
	service, err := r.serviceResolver.ResolvePodToServiceIdentity(ctx, pod)
	callTime := time.Since(startTime)
	logrus.Debugf("Service resolver took %s pod %s service %s", callTime, pod.Name, service.Name)
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

func (r *ipResolverImpl) ResolveDNS(dns string, captureTime time.Time) (Identity, error) {
	if !strings.HasSuffix(dns, viper.GetString(config.ClusterDomainKey)) {
		logrus.Debugf("DNS %s does not belong to cluster domain, ignoring", dns)
		return Identity{}, NotK8sService
	}

	startTime := time.Now()
	ips, err := r.finder.ResolveServiceAddressToIps(context.Background(), dns)
	callTime := time.Since(startTime)
	logrus.Debugf("DNS resolver took %s dns %s ips %v", callTime, dns, ips)
	if err != nil {
		return Identity{}, err
	}

	if len(ips) == 0 {
		logrus.Debugf("Service address %s is currently not backed by any pod, ignoring", dns)
		return Identity{}, NotPodAddress
	}

	// TODO: Explain why we are using the first IP
	serviceName, err := r.ResolveIp(ips[0], captureTime)
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
