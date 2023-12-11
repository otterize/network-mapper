package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/prometheus"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

func (r *mutationResolver) ResetCapture(ctx context.Context) (bool, error) {
	logrus.Info("Resetting stored intents")
	r.intentsHolder.Reset()
	return true, nil
}

func (r *mutationResolver) ReportCaptureResults(ctx context.Context, results model.CaptureResults) (bool, error) {
	select {
	case r.dnsCaptureResults <- results:
		prometheus.IncrementDNSCaptureReports(len(results.Results))
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		prometheus.IncrementDNSCaptureDrops(len(results.Results))
		return false, nil
	}
}

func (r *mutationResolver) ReportSocketScanResults(ctx context.Context, results model.SocketScanResults) (bool, error) {
	select {
	case r.socketScanResults <- results:
		prometheus.IncrementSocketScanReports(len(results.Results))
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		prometheus.IncrementSocketScanDrops(len(results.Results))
		return false, nil
	}
}

func (r *mutationResolver) ReportKafkaMapperResults(ctx context.Context, results model.KafkaMapperResults) (bool, error) {
	select {
	case r.kafkaMapperResults <- results:
		prometheus.IncrementKafkaReports(len(results.Results))
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		prometheus.IncrementKafkaDrops(len(results.Results))
		return false, nil
	}
}

func (r *mutationResolver) ReportIstioConnectionResults(ctx context.Context, results model.IstioConnectionResults) (bool, error) {
	select {
	case r.istioConnectionResults <- results:
		prometheus.IncrementIstioReports(len(results.Results))
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		prometheus.IncrementIstioDrops(len(results.Results))
		return false, nil
	}
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
