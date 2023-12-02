package resolvers

import (
	"context"
	"errors"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/prometheus"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
)

type SourceType string

const (
	SourceTypeDNSCapture  SourceType = "Capture"
	SourceTypeSocketScan  SourceType = "SocketScan"
	SourceTypeKafkaMapper SourceType = "KafkaMapper"
	SourceTypeIstio       SourceType = "Istio"
)

func updateTelemetriesCounters(sourceType SourceType, intent model.Intent) {
	clientKey := telemetrysender.Anonymize(fmt.Sprintf("%s/%s", intent.Client.Namespace, intent.Client.Name))
	serverKey := telemetrysender.Anonymize(fmt.Sprintf("%s/%s", intent.Server.Namespace, intent.Server.Name))
	intentKey := telemetrysender.Anonymize(fmt.Sprintf("%s-%s", clientKey, serverKey))

	telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeIntentsDiscovered, intentKey)
	telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeServiceDiscovered, clientKey)
	telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeServiceDiscovered, serverKey)

	telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeNamespaceDiscovered, telemetrysender.Anonymize(intent.Client.Namespace))
	telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeNamespaceDiscovered, telemetrysender.Anonymize(intent.Server.Namespace))

	if sourceType == SourceTypeDNSCapture {
		telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeIntentsDiscoveredCapture, intentKey)
	} else if sourceType == SourceTypeSocketScan {
		telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeIntentsDiscoveredSocketScan, intentKey)
	} else if sourceType == SourceTypeKafkaMapper {
		telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeIntentsDiscoveredKafka, intentKey)
	} else if sourceType == SourceTypeIstio {
		telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeIntentsDiscoveredIstio, intentKey)
	}
}

func (r *mutationResolver) tryHandleSocketScanDestinationAsService(ctx context.Context, srcSvcIdentity model.OtterizeServiceIdentity, dest model.Destination) (bool, error) {
	destSvc, foundSvc, err := r.kubeFinder.ResolveIPToService(ctx, dest.Destination)
	if err != nil {
		return false, err
	}
	if !foundSvc {
		return false, nil
	}
	err = r.addSocketScanServiceIntent(ctx, srcSvcIdentity, dest, destSvc)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) addSocketScanServiceIntent(ctx context.Context, srcSvcIdentity model.OtterizeServiceIdentity, dest model.Destination, svc *corev1.Service) error {
	pods, err := r.kubeFinder.ResolveServiceToPods(ctx, svc)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		logrus.Debugf("could not find any pods for service '%s' in namespace '%s'", svc.Name, svc.Namespace)
		return nil
	}

	// Assume the pods backing the service are identical
	pod := pods[0]

	if pod.CreationTimestamp.After(dest.LastSeen) {
		logrus.Debugf("Pod %s was created after scan time %s, ignoring", pod.Name, dest.LastSeen)
		return nil
	}

	dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, &pod)
	if err != nil {
		return err
	}

	dstSvcIdentity := &model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: pod.Namespace, Labels: podLabelsToOtterizeLabels(&pod)}
	if dstService.OwnerObject != nil {
		dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
	}
	dstSvcIdentity.KubernetesService = lo.ToPtr(svc.Name)

	intent := model.Intent{
		Client: &srcSvcIdentity,
		Server: dstSvcIdentity,
	}

	r.intentsHolder.AddIntent(
		dest.LastSeen,
		intent,
	)

	updateTelemetriesCounters(SourceTypeSocketScan, intent)
	prometheus.IncrementSocketScanReports(1)
	return nil
}

func (r *mutationResolver) addSocketScanPodIntent(ctx context.Context, srcSvcIdentity model.OtterizeServiceIdentity, dest model.Destination, destPod *corev1.Pod) error {
	if destPod.DeletionTimestamp != nil {
		logrus.Debugf("Pod %s is being deleted, ignoring", destPod.Name)
		return nil
	}

	if destPod.CreationTimestamp.After(dest.LastSeen) {
		logrus.Debugf("Pod %s was created after scan time %s, ignoring", destPod.Name, dest.LastSeen)
		return nil
	}

	dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
	if err != nil {
		return err
	}

	dstSvcIdentity := &model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: destPod.Namespace, Labels: podLabelsToOtterizeLabels(destPod)}
	if dstService.OwnerObject != nil {
		dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
	}

	intent := model.Intent{
		Client: &srcSvcIdentity,
		Server: dstSvcIdentity,
	}

	r.intentsHolder.AddIntent(
		dest.LastSeen,
		intent,
	)
	updateTelemetriesCounters(SourceTypeSocketScan, intent)
	prometheus.IncrementSocketScanReports(1)
	return nil
}

func (r *mutationResolver) handleDNSCaptureResultsAsExternalTraffic(_ context.Context, dest model.Destination, srcSvcIdentity model.OtterizeServiceIdentity) error {
	if !viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
		return nil
	}
	intent := ExternalTrafficIntent{
		Client:   srcSvcIdentity,
		LastSeen: dest.LastSeen,
		DNSName:  dest.Destination,
	}
	ip := "(unknown)"
	if dest.DestinationIP != nil {
		ip = *dest.DestinationIP
		intent.IPs = map[IP]struct{}{IP(*dest.DestinationIP): {}}
	}
	logrus.Debugf("Saw external traffic, from '%s.%s' to '%s' (IP '%s')", srcSvcIdentity.Name, srcSvcIdentity.Namespace, dest.Destination, ip)

	r.externalTrafficIntentsHolder.AddIntent(intent)
	return nil
}

func (r *mutationResolver) handleDNSCaptureResultsAsKubernetesPods(ctx context.Context, dest model.Destination, srcSvcIdentity model.OtterizeServiceIdentity) error {
	destAddress := dest.Destination
	ips, serviceName, err := r.kubeFinder.ResolveServiceAddressToIps(ctx, destAddress)
	if err != nil {
		logrus.WithError(err).Warningf("Could not resolve service address %s", destAddress)
		return nil
	}
	if len(ips) == 0 {
		logrus.Debugf("Service address %s is currently not backed by any pod, ignoring", destAddress)
		return nil
	}
	destPod, err := r.kubeFinder.ResolveIPToPod(ctx, ips[0])
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", ips[0])
		} else {
			logrus.WithError(err).Debugf("Could not resolve %s to pod", ips[0])
		}
		return nil
	}

	if destPod.CreationTimestamp.After(dest.LastSeen) {
		logrus.Debugf("Pod %s was created after capture time %s, ignoring", destPod.Name, dest.LastSeen)
		return nil
	}

	if destPod.DeletionTimestamp != nil {
		logrus.Debugf("Pod %s is being deleted, ignoring", destPod.Name)
		return nil
	}

	dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
	if err != nil {
		logrus.WithError(err).Debugf("Could not resolve pod %s to identity", destPod.Name)
		return nil
	}

	dstSvcIdentity := &model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: destPod.Namespace, Labels: podLabelsToOtterizeLabels(destPod)}
	if dstService.OwnerObject != nil {
		dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
	}
	if serviceName != "" {
		dstSvcIdentity.KubernetesService = lo.ToPtr(serviceName)
	}

	intent := model.Intent{
		Client: &srcSvcIdentity,
		Server: dstSvcIdentity,
	}

	r.intentsHolder.AddIntent(
		dest.LastSeen,
		intent,
	)
	updateTelemetriesCounters(SourceTypeDNSCapture, intent)

	return nil
}
