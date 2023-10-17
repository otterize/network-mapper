package resolvers

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/prometheus"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
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

	// Dummy comment to trigger CICD
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

func (r *mutationResolver) tryHandleSocketScanDestinationAsService(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, dest model.Destination) (bool, error) {
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

func (r *mutationResolver) addSocketScanServiceIntent(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, dest model.Destination, svc *corev1.Service) error {
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
		Client: srcSvcIdentity,
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
func (r *mutationResolver) addSocketScanPodIntent(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, dest model.Destination, destPod *corev1.Pod) error {
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
		Client: srcSvcIdentity,
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
