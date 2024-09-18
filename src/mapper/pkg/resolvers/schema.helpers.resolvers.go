package resolvers

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/incomingtrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/prometheus"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"time"
)

type SourceType string

const (
	SourceTypeDNSCapture  SourceType = "Capture"
	SourceTypeTCPScan     SourceType = "TCPScan"
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
	} else if sourceType == SourceTypeTCPScan {
		telemetrysender.IncrementUniqueCounterNetworkMapper(telemetriesgql.EventTypeIntentsDiscoveredTcp, intentKey)
	}
}

func (r *Resolver) resolveDestIdentity(ctx context.Context, dest model.Destination, lastSeen time.Time) (model.OtterizeServiceIdentity, bool, error) {
	destSvc, foundSvc, err := r.kubeFinder.ResolveIPToService(ctx, dest.Destination)
	if err != nil {
		return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
	}
	if foundSvc {
		dstSvcIdentity, ok, err := r.kubeFinder.ResolveOtterizeIdentityForService(ctx, destSvc, lastSeen)
		if err != nil {
			return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
		}
		if ok {
			return dstSvcIdentity, true, nil
		}
	}

	destPod, err := r.kubeFinder.ResolveIPToPod(ctx, dest.Destination)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", dest.Destination)
		} else {
			logrus.WithError(err).Debugf("Could not resolve %s to pod", dest.Destination)
		}
		return model.OtterizeServiceIdentity{}, false, nil
	}

	if destPod.CreationTimestamp.After(dest.LastSeen) {
		logrus.Debugf("Pod %s was created after capture time %s, ignoring", destPod.Name, dest.LastSeen)
		return model.OtterizeServiceIdentity{}, false, nil
	}

	if destPod.DeletionTimestamp != nil {
		logrus.Debugf("Pod %s is being deleted, ignoring", destPod.Name)
		return model.OtterizeServiceIdentity{}, false, nil
	}

	dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
	if err != nil {
		logrus.WithError(err).Debugf("Could not resolve pod %s to identity", destPod.Name)
		return model.OtterizeServiceIdentity{}, false, nil
	}

	dstSvcIdentity := model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: destPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(destPod)}
	if dstService.OwnerObject != nil {
		dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
	}

	return dstSvcIdentity, true, nil
}

func (r *Resolver) tryHandleSocketScanDestinationAsService(ctx context.Context, srcSvcIdentity model.OtterizeServiceIdentity, dest model.Destination) (bool, error) {
	destSvc, foundSvc, err := r.kubeFinder.ResolveIPToService(ctx, dest.Destination)
	if err != nil {
		return false, errors.Wrap(err)
	}
	if !foundSvc {
		return false, nil
	}
	err = r.addSocketScanServiceIntent(ctx, srcSvcIdentity, dest, destSvc)
	if err != nil {
		return false, errors.Wrap(err)
	}
	return true, nil
}

func (r *Resolver) addSocketScanServiceIntent(ctx context.Context, srcSvcIdentity model.OtterizeServiceIdentity, dest model.Destination, svc *corev1.Service) error {
	lastSeen := dest.LastSeen
	dstSvcIdentity, ok, err := r.kubeFinder.ResolveOtterizeIdentityForService(ctx, svc, dest.LastSeen)
	if err != nil {
		return errors.Wrap(err)
	}

	if !ok {
		return nil
	}

	intent := model.Intent{
		Client: &srcSvcIdentity,
		Server: &dstSvcIdentity,
	}

	r.intentsHolder.AddIntent(
		lastSeen,
		intent,
	)

	updateTelemetriesCounters(SourceTypeSocketScan, intent)
	prometheus.IncrementSocketScanReports(1)
	return nil
}

func (r *Resolver) addSocketScanPodIntent(ctx context.Context, srcSvcIdentity model.OtterizeServiceIdentity, dest model.Destination, destPod *corev1.Pod) error {
	if destPod.DeletionTimestamp != nil {
		logrus.Debugf("Pod %s is being deleted, ignoring", destPod.Name)
		return nil
	}

	minTimeForPodCreationTime := dest.LastSeen.Add(-viper.GetDuration(config.TimeServerHasToLiveBeforeWeTrustItKey))
	if destPod.CreationTimestamp.After(minTimeForPodCreationTime) {
		logrus.Debugf("Pod %s was created after scan time %s, ignoring", destPod.Name, minTimeForPodCreationTime)
		return nil
	}

	dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
	if err != nil {
		return errors.Wrap(err)
	}

	dstSvcIdentity := &model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: destPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(destPod)}
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

func (r *Resolver) handleDNSCaptureResultsAsExternalTraffic(_ context.Context, dest model.Destination, srcSvcIdentity model.OtterizeServiceIdentity) error {
	if !viper.GetBool(config.ExternalTrafficCaptureEnabledKey) {
		return nil
	}
	intent := externaltrafficholder.ExternalTrafficIntent{
		Client:   srcSvcIdentity,
		LastSeen: dest.LastSeen,
		DNSName:  dest.Destination,
	}
	ip := "(unknown)"
	if dest.DestinationIP != nil {
		ip = *dest.DestinationIP
		intent.IPs = map[externaltrafficholder.IP]struct{}{externaltrafficholder.IP(*dest.DestinationIP): {}}
		r.dnsCache.AddOrUpdateDNSData(dest.Destination, ip, int(lo.FromPtr(dest.TTL)))
	}
	logrus.Debugf("Saw external traffic, from '%s.%s' to '%s' (IP '%s')", srcSvcIdentity.Name, srcSvcIdentity.Namespace, dest.Destination, ip)

	r.externalTrafficIntentsHolder.AddIntent(intent)
	return nil
}

// ReportAWSOperation is the resolver for the reportAWSOperation field.
func (r *Resolver) handleAWSOperationReport(ctx context.Context, operation model.AWSOperationResults) error {
	for _, op := range operation {
		logrus.Debugf("Received AWS operation: %+v", op)
		srcPod, err := r.kubeFinder.ResolveIPToPod(ctx, op.SrcIP)

		if err != nil {
			logrus.Errorf("could not resolve %s to pod: %s", op.SrcIP, err.Error())
			continue
		}

		serviceId, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)

		if err != nil {
			logrus.Errorf("could not resolve pod %s to identity: %s", srcPod.Name, err.Error())
			continue
		}

		r.awsIntentsHolder.AddIntent(awsintentsholder.AWSIntent{
			Client: model.OtterizeServiceIdentity{
				Name:      serviceId.Name,
				Namespace: srcPod.Namespace,
			},
			Actions: op.Actions,
			ARN:     op.Resource,
		})

		logrus.
			WithField("client", serviceId.Name).
			WithField("namespace", srcPod.Namespace).
			WithField("actions", op.Actions).
			WithField("arn", op.Resource).
			Debug("Discovered AWS intent")
	}

	return nil
}

func (r *Resolver) handleDNSCaptureResultsAsKubernetesPods(ctx context.Context, dest model.Destination, srcSvcIdentity model.OtterizeServiceIdentity) error {
	dstSvcIdentity, ok, err := r.resolveOtterizeIdentityForDestinationAddress(ctx, dest)
	if err != nil {
		return errors.Wrap(err)
	}
	if !ok {
		return nil
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

func (r *Resolver) resolveOtterizeIdentityForDestinationAddress(ctx context.Context, dest model.Destination) (*model.OtterizeServiceIdentity, bool, error) {
	destAddress := dest.Destination
	pods, serviceName, err := r.kubeFinder.ResolveServiceAddressToPods(ctx, destAddress)
	if err != nil {
		logrus.WithError(err).Warningf("Could not resolve service address %s", destAddress)
		// Intentionally no error return
		return nil, false, nil
	}
	if kubefinder.ServiceIsAPIServer(serviceName.Name, serviceName.Namespace) {
		return &model.OtterizeServiceIdentity{
			Name:              serviceName.Name,
			Namespace:         serviceName.Namespace,
			KubernetesService: &serviceName.Name,
		}, true, nil
	}

	filteredPods := lo.Filter(pods, func(pod corev1.Pod, _ int) bool {
		if pod.Spec.HostNetwork {
			logrus.Debugf("pod %s is in host network, ignoring", pod.Name)
			return false
		}
		lastCreationTimeForUsToTrustIt := dest.LastSeen
		if lo.IsEmpty(serviceName) {
			// In this case the DNS was a "pod" DNS - which contains IP - and therefore less reliable.
			lastCreationTimeForUsToTrustIt = lastCreationTimeForUsToTrustIt.Add(viper.GetDuration(config.TimeServerHasToLiveBeforeWeTrustItKey))
		}
		return lastCreationTimeForUsToTrustIt.After(pod.CreationTimestamp.Time) && pod.DeletionTimestamp == nil
	})

	if len(filteredPods) == 0 {
		logrus.Debugf("Service address %s is currently not backed by any valid pod, ignoring", destAddress)
		return nil, false, nil
	}

	destPod := &filteredPods[0]

	dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
	if err != nil {
		logrus.WithError(err).Debugf("Could not resolve pod %s to identity", destPod.Name)
		return nil, false, nil
	}

	dstSvcIdentity := &model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: destPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(destPod)}
	if dstService.OwnerObject != nil {
		dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
	}
	if serviceName.Name != "" {
		dstSvcIdentity.KubernetesService = &serviceName.Name
	}
	return dstSvcIdentity, true, nil
}

func (r *Resolver) resolveOtterizeIdentityForExternalAccessDestination(ctx context.Context, dest model.Destination) (model.OtterizeServiceIdentity, bool, error) {
	destIP := lo.FromPtr(dest.DestinationIP)
	if dest.DestinationIP == nil || len(destIP) == 0 {
		return model.OtterizeServiceIdentity{}, false, errors.New("invalid TCP destination, IP is empty")
	}
	if dest.DestinationPort == nil {
		return model.OtterizeServiceIdentity{}, false, errors.New("invalid TCP destination, port is empty")
	}

	destPort := int(*dest.DestinationPort)
	destService, ok, err := r.kubeFinder.ResolveIPToExternalAccessService(ctx, destIP, destPort)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", destIP)
			return model.OtterizeServiceIdentity{}, false, nil
		}
		logrus.WithError(err).Debugf("Could not resolve %s to pod", destIP)
		return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
	}
	if !ok {
		// If the traffic is not to a NodePort or LoadBalancer Service, it can be traffic from a loadbalancer like AWS ALB
		// to a pod.
		pod, err := r.kubeFinder.ResolveIPToPod(ctx, destIP)
		if err != nil {
			if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
				logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", destIP)
				return model.OtterizeServiceIdentity{}, false, nil
			}
			if errors.Is(err, kubefinder.ErrNoPodFound) {
				logrus.WithError(err).Debugf("Could not resolve %s to pod: no pod found", destIP)
				return model.OtterizeServiceIdentity{}, false, nil
			}
			logrus.WithError(err).Debugf("Could not resolve %s to pod", destIP)
			return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
		}
		dstSvcIdentity, err := r.resolveInClusterIdentity(ctx, pod)
		if err != nil {
			return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
		}
		return dstSvcIdentity, true, nil
	}

	dstSvcIdentity, ok, err := r.kubeFinder.ResolveOtterizeIdentityForService(ctx, destService, dest.LastSeen)
	if err != nil {
		return model.OtterizeServiceIdentity{}, false, errors.Wrap(err)
	}
	if !ok {
		return model.OtterizeServiceIdentity{}, false, nil
	}

	return dstSvcIdentity, true, nil
}

func (r *Resolver) handleReportTCPCaptureResults(ctx context.Context, results model.CaptureTCPResults) error {
	if !viper.GetBool(sharedconfig.EnableTCPKey) {
		return nil
	}

	for _, captureItem := range results.Results {
		err := r.handleTCPCaptureResult(ctx, captureItem)
		if err != nil {
			logrus.WithError(err).
				WithField("srcIp", captureItem.SrcIP).
				WithField("srcHostname", captureItem.SrcHostname).
				Error("could not handle TCP capture result")
		}
	}
	telemetrysender.SendNetworkMapper(telemetriesgql.EventTypeIntentsDiscoveredCapture, len(results.Results))
	r.gotResultsSignal()
	return nil
}

func (r *Resolver) handleTCPCaptureResult(ctx context.Context, captureItem model.RecordedDestinationsForSrc) error {
	logrus.Debugf("Handling TCP capture result from %s to %s:%d", captureItem.SrcIP, captureItem.Destinations[0].Destination, lo.FromPtr(captureItem.Destinations[0].DestinationPort))
	isSrcInCluster, err := r.kubeFinder.IsSrcIpClusterInternal(ctx, captureItem.SrcIP)
	if err != nil {
		return errors.Wrap(err)
	}
	if !isSrcInCluster {
		return errors.Wrap(r.reportIncomingInternetTraffic(ctx, captureItem.SrcIP, captureItem.Destinations))
	}

	srcSvcIdentity, err := r.discoverInternalSrcIdentity(ctx, captureItem)
	if err != nil {
		logrus.WithError(err).Debugf("could not discover src identity for '%s'", captureItem.SrcIP)
		return nil
	}
	for _, dest := range captureItem.Destinations {
		r.handleInternalTrafficTCPResult(ctx, srcSvcIdentity, dest)
	}
	return nil
}

func (r *Resolver) reportIncomingInternetTraffic(ctx context.Context, srcIP string, destinations []model.Destination) error {
	for _, dest := range destinations {
		destSvcIdentity, ok, err := r.resolveOtterizeIdentityForExternalAccessDestination(ctx, dest)
		if err != nil {
			logrus.WithError(err).Error("could not resolve incoming destination identity")
			continue
		}
		if !ok {
			continue
		}

		logrus.Debugf("Saw incoming traffic from '%s' to '%s/%s'", srcIP, destSvcIdentity.Name, destSvcIdentity.Namespace)
		intent := incomingtrafficholder.IncomingTrafficIntent{
			LastSeen: dest.LastSeen,
			Server:   destSvcIdentity,
			IP:       srcIP,
		}
		r.incomingTrafficHolder.AddIntent(intent)
	}
	return nil
}

func (r *Resolver) handleInternalTrafficTCPResult(ctx context.Context, srcIdentity model.OtterizeServiceIdentity, dest model.Destination) {
	lastSeen := dest.LastSeen
	destIdentity, ok, err := r.resolveDestIdentity(ctx, dest, lastSeen)
	if err != nil {
		logrus.WithError(err).Error("could not resolve destination identity")
		return
	}
	if !ok {
		return
	}

	intent := model.Intent{
		Client: &srcIdentity,
		Server: &destIdentity,
	}

	r.intentsHolder.AddIntent(
		dest.LastSeen,
		intent,
	)
	updateTelemetriesCounters(SourceTypeTCPScan, intent)
}

func (r *Resolver) handleReportCaptureResults(ctx context.Context, results model.CaptureResults) error {
	if !viper.GetBool(sharedconfig.EnableDNSKey) {
		return nil
	}

	var newResults int
	for _, captureItem := range results.Results {
		srcSvcIdentity, err := r.discoverInternalSrcIdentity(ctx, captureItem)
		if err != nil {
			logrus.WithError(err).Debugf("could not discover src identity for '%s'", captureItem.SrcIP)
			continue
		}
		for _, dest := range captureItem.Destinations {
			destCopy := dest
			destAddress := dest.Destination
			if !strings.HasSuffix(destAddress, viper.GetString(config.ClusterDomainKey)) {
				err := r.handleDNSCaptureResultsAsExternalTraffic(ctx, destCopy, srcSvcIdentity)
				if err != nil {
					logrus.WithError(err).Error("could not handle DNS capture result as external traffic")
					continue
				}
				newResults++
				continue
			}

			err := r.handleDNSCaptureResultsAsKubernetesPods(ctx, destCopy, srcSvcIdentity)
			if err != nil {
				logrus.WithError(err).Error("could not handle DNS capture result as pod")
				continue
			}
			newResults++
		}
	}

	prometheus.IncrementDNSCaptureReports(newResults)
	r.gotResultsSignal()
	return nil
}

func (r *Resolver) handleReportSocketScanResults(ctx context.Context, results model.SocketScanResults) error {
	if !viper.GetBool(sharedconfig.EnableSocketScannerKey) {
		return nil
	}
	for _, socketScanItem := range results.Results {
		srcSvcIdentity, err := r.discoverInternalSrcIdentity(ctx, socketScanItem)
		if err != nil {
			logrus.WithError(err).Debugf("could not discover src identity for '%s'", socketScanItem.SrcIP)
			continue
		}
		for _, dest := range socketScanItem.Destinations {
			destCopy := dest
			isService, err := r.tryHandleSocketScanDestinationAsService(ctx, srcSvcIdentity, destCopy)
			if err != nil {
				logrus.WithError(err).Errorf("failed to handle IP '%s' as service, it may or may not be a service. This error only occurs if something failed; not if the IP does not belong to a service.", dest.Destination)
				// Log error but don't stop handling other destinations.
				continue
			}

			if isService {
				continue // No need to try to handle IP as Pod, since IP belonged to a service.
			}

			destPod, err := r.kubeFinder.ResolveIPToPod(ctx, destCopy.Destination)
			if err != nil {
				logrus.WithError(err).Debugf("Could not resolve %s to pod", dest.Destination)
				// Log error but don't stop handling other destinations.
				continue
			}

			err = r.addSocketScanPodIntent(ctx, srcSvcIdentity, destCopy, destPod)
			if err != nil {
				logrus.WithError(err).Errorf("failed to resolve IP '%s' to pod", dest.Destination)
				// Log error but don't stop handling other destinations.
				continue
			}
		}
	}
	r.gotResultsSignal()
	return nil
}

func (r *Resolver) handleReportKafkaMapperResults(ctx context.Context, results model.KafkaMapperResults) error {
	var newResults int
	for _, result := range results.Results {
		srcPod, err := r.kubeFinder.ResolveIPToPod(ctx, result.SrcIP)
		if err != nil {
			if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
				logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", result.SrcIP)
			} else {
				logrus.WithError(err).Debugf("Could not resolve %s to pod", result.SrcIP)
			}
			continue
		}

		if srcPod.DeletionTimestamp != nil {
			logrus.Debugf("Pod %s is being deleted, ignoring", srcPod.Name)
			continue
		}

		if srcPod.CreationTimestamp.After(result.LastSeen) {
			logrus.Debugf("Pod %s was created after scan time %s, ignoring", srcPod.Name, result.LastSeen)
			continue
		}

		srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", srcPod.Name)
			continue
		}

		srcSvcIdentity := model.OtterizeServiceIdentity{Name: srcService.Name, Namespace: srcPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(srcPod)}

		dstPod, err := r.kubeFinder.ResolvePodByName(ctx, result.ServerPodName, result.ServerNamespace)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", result.ServerPodName)
			continue
		}
		dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, dstPod)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", dstPod.Name)
			continue
		}
		dstSvcIdentity := model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: dstPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(dstPod)}

		operation, err := model.KafkaOpFromText(result.Operation)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve kafka operation %s", result.Operation)
			return err
		}

		intent := model.Intent{
			Client: &srcSvcIdentity,
			Server: &dstSvcIdentity,
			Type:   lo.ToPtr(model.IntentTypeKafka),
			KafkaTopics: []model.KafkaConfig{
				{
					Name:       result.Topic,
					Operations: []model.KafkaOperation{operation},
				},
			},
		}

		updateTelemetriesCounters(SourceTypeKafkaMapper, intent)
		r.intentsHolder.AddIntent(
			result.LastSeen,
			intent,
		)
		newResults++
	}

	prometheus.IncrementKafkaReports(newResults)
	r.gotResultsSignal()
	return nil
}

func (r *Resolver) handleReportIstioConnectionResults(ctx context.Context, results model.IstioConnectionResults) error {
	var newResults int
	for _, result := range results.Results {
		srcPod, err := r.kubeFinder.ResolveIstioWorkloadToPod(ctx, result.SrcWorkload, result.SrcWorkloadNamespace)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve workload %s to pod", result.SrcWorkload)
			continue
		}
		dstPod, err := r.kubeFinder.ResolveIstioWorkloadToPod(ctx, result.DstWorkload, result.DstWorkloadNamespace)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve workload %s to pod", result.DstWorkload)
			continue
		}
		srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", srcPod.Name)
			continue
		}
		dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, dstPod)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", dstPod.Name)
			continue
		}

		srcSvcIdentity := model.OtterizeServiceIdentity{Name: srcService.Name, Namespace: srcPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(srcPod)}
		dstSvcIdentity := model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: dstPod.Namespace, Labels: kubefinder.PodLabelsToOtterizeLabels(dstPod)}
		if srcService.OwnerObject != nil {
			srcSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(srcService.OwnerObject.GetObjectKind().GroupVersionKind())
		}

		if dstService.OwnerObject != nil {
			dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
			if result.DstServiceName != "" {
				dstSvcIdentity.KubernetesService = &result.DstServiceName
			}
		}

		intent := model.Intent{
			Client:        &srcSvcIdentity,
			Server:        &dstSvcIdentity,
			Type:          lo.ToPtr(model.IntentTypeHTTP),
			HTTPResources: []model.HTTPResource{{Path: result.Path, Methods: result.Methods}},
		}

		updateTelemetriesCounters(SourceTypeIstio, intent)
		r.intentsHolder.AddIntent(result.LastSeen, intent)
		newResults++
	}

	prometheus.IncrementIstioReports(newResults)
	r.gotResultsSignal()
	return nil
}

type Results interface {
	Length() int
}

type resultsHandlerFunc[T Results] func(ctx context.Context, results T) error

func runHandleLoop[T Results](ctx context.Context, resultsChan chan T, handleFunc resultsHandlerFunc[T]) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case results := <-resultsChan:
			err := handleFunc(ctx, results)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to handle %d results of type '%T'", results.Length(), results)
				// Intentionally no return
			}
		}
	}
}
