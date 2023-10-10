package resolvers

import (
	"context"
	"errors"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetriesgql"
	"github.com/otterize/intents-operator/src/shared/telemetries/telemetrysender"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/prometheus"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

func (r *mutationResolver) handleSocketScanService(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, dest model.Destination, svc *corev1.Service) error {
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
func (r *mutationResolver) handleSocketScanPod(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, dest model.Destination, destPod *corev1.Pod) error {
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

func (r *mutationResolver) discoverIntentFromSocketScan(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, destIp model.Destination) {
	err := r.discoverIntentFromService(ctx, srcSvcIdentity, destIp)
	if err != nil {
		return
	}

	r.discoverIntentFromPod(ctx, srcSvcIdentity, destIp)
}

func (r *mutationResolver) discoverIntentFromPod(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, destIp model.Destination) {
	destPod, err := r.kubeFinder.ResolveIPToPod(ctx, destIp.Destination)
	if err != nil {
		if errors.Is(err, kubefinder.ErrFoundMoreThanOnePod) {
			logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", destIp.Destination)
		} else {
			logrus.WithError(err).Debugf("Could not resolve %s to pod", destIp.Destination)
		}
		return
	}

	err = r.handleSocketScanPod(ctx, srcSvcIdentity, destIp, destPod)
	if err != nil {
		logrus.WithError(err).Errorf("failed to resolve IP '%s' to pod", destIp.Destination)
	}
}

func (r *mutationResolver) discoverIntentFromService(ctx context.Context, srcSvcIdentity *model.OtterizeServiceIdentity, destIp model.Destination) error {
	destSvc, err := r.kubeFinder.ResolveIPToService(ctx, destIp.Destination)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logrus.Debugf("IP %s is not backed by any service, ignoring", destIp.Destination)
			return nil
		}
		logrus.WithError(err).Errorf("Could not resolve IP '%s' to service", destIp)
		return err
	}

	err = r.handleSocketScanService(ctx, srcSvcIdentity, destIp, destSvc)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to handle service '%s' in namespace '%s'", destSvc.Name, destSvc.Namespace)
		return err
	}

	return nil
}
