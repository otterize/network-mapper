package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"strings"

	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/mapper/pkg/prometheus"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

func (r *mutationResolver) ResetCapture(ctx context.Context) (bool, error) {
	logrus.Info("Resetting stored intents")
	r.intentsHolder.Reset()
	return true, nil
}

func (r *mutationResolver) ReportCaptureResults(ctx context.Context, results model.CaptureResults) (bool, error) {
	var newResults int
	for _, captureItem := range results.Results {
		srcSvcIdentity, err := r.discoverSrcIdentity(ctx, captureItem)
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
	return true, nil
}

func (r *mutationResolver) ReportSocketScanResults(ctx context.Context, results model.SocketScanResults) (bool, error) {
	for _, socketScanItem := range results.Results {
		srcSvcIdentity, err := r.discoverSrcIdentity(ctx, socketScanItem)
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
	return true, nil
}

func (r *mutationResolver) ReportKafkaMapperResults(ctx context.Context, results model.KafkaMapperResults) (bool, error) {
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

		srcSvcIdentity := model.OtterizeServiceIdentity{Name: srcService.Name, Namespace: srcPod.Namespace, Labels: podLabelsToOtterizeLabels(srcPod)}

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
		dstSvcIdentity := model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: dstPod.Namespace, Labels: podLabelsToOtterizeLabels(dstPod)}

		operation, err := model.KafkaOpFromText(result.Operation)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve kafka operation %s", result.Operation)
			return false, err
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
	return true, nil
}

func (r *mutationResolver) ReportIstioConnectionResults(ctx context.Context, results model.IstioConnectionResults) (bool, error) {
	var newResults int
	for _, result := range results.Results {
		srcPod, err := r.kubeFinder.ResolveIstioWorkloadToPod(ctx, result.SrcWorkload, result.SrcWorkloadNamespace)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve workload %s to pod", result.SrcWorkload)
			continue
		}
		dstPod, err := r.kubeFinder.ResolveIstioWorkloadToPod(ctx, result.DstWorkload, result.DstWorkloadNamespace)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve workload %s to pod", result.SrcWorkload)
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

		srcSvcIdentity := model.OtterizeServiceIdentity{Name: srcService.Name, Namespace: srcPod.Namespace, Labels: podLabelsToOtterizeLabels(srcPod)}
		dstSvcIdentity := model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: dstPod.Namespace, Labels: podLabelsToOtterizeLabels(dstPod)}
		if srcService.OwnerObject != nil {
			srcSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(srcService.OwnerObject.GetObjectKind().GroupVersionKind())
		}

		if dstService.OwnerObject != nil {
			dstSvcIdentity.PodOwnerKind = model.GroupVersionKindFromKubeGVK(dstService.OwnerObject.GetObjectKind().GroupVersionKind())
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
	return true, nil
}

func (r *queryResolver) ServiceIntents(ctx context.Context, namespaces []string, includeLabels []string, includeAllLabels *bool) ([]model.ServiceIntents, error) {
	shouldIncludeAllLabels := false
	if includeAllLabels != nil && *includeAllLabels {
		shouldIncludeAllLabels = true
	}
	discoveredIntents, err := r.intentsHolder.GetIntents(namespaces, includeLabels, []string{}, shouldIncludeAllLabels, nil)
	if err != nil {
		return []model.ServiceIntents{}, err
	}
	intentsBySource := intentsstore.GroupIntentsBySource(discoveredIntents)

	// sorting by service name so results are more consistent
	slices.SortFunc(intentsBySource, func(intentsa, intentsb model.ServiceIntents) bool {
		return intentsa.Client.AsNamespacedName().String() < intentsb.Client.AsNamespacedName().String()
	})

	for _, intents := range intentsBySource {
		slices.SortFunc(intents.Intents, func(desta, destb model.OtterizeServiceIdentity) bool {
			return desta.AsNamespacedName().String() < destb.AsNamespacedName().String()
		})
	}

	return intentsBySource, nil
}

func (r *queryResolver) Intents(ctx context.Context, namespaces []string, includeLabels []string, excludeServiceWithLabels []string, includeAllLabels *bool, server *model.ServerFilter) ([]model.Intent, error) {
	shouldIncludeAllLabels := false
	if includeAllLabels != nil && *includeAllLabels {
		shouldIncludeAllLabels = true
	}

	timestampedIntents, err := r.intentsHolder.GetIntents(
		namespaces,
		includeLabels,
		excludeServiceWithLabels,
		shouldIncludeAllLabels,
		server,
	)
	if err != nil {
		return []model.Intent{}, err
	}

	intents := lo.Map(timestampedIntents, func(timestampedIntent intentsstore.TimestampedIntent, _ int) model.Intent {
		return timestampedIntent.Intent
	})

	// sort by service names for consistent ordering
	slices.SortFunc(intents, func(intenta, intentb model.Intent) bool {
		clienta, clientb := intenta.Client.AsNamespacedName(), intentb.Client.AsNamespacedName()
		servera, serverb := intenta.Server.AsNamespacedName(), intentb.Server.AsNamespacedName()

		if clienta != clientb {
			return clienta.String() < clientb.String()
		}

		return servera.String() < serverb.String()
	})

	return intents, nil
}

func (r *queryResolver) Health(ctx context.Context) (bool, error) {
	return true, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
