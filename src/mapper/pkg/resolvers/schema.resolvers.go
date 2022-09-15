package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/otterize/network-mapper/mapper/pkg/config"
	"github.com/otterize/network-mapper/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/mapper/pkg/kubefinder"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func (r *mutationResolver) ResetCapture(ctx context.Context) (bool, error) {
	r.intentsHolder.Reset()
	return true, nil
}

func (r *mutationResolver) ReportCaptureResults(ctx context.Context, results model.CaptureResults) (bool, error) {
	for _, captureItem := range results.Results {
		srcPod, err := r.kubeFinder.ResolveIpToPod(ctx, captureItem.SrcIP)
		if err != nil {
			if errors.Is(err, kubefinder.FoundMoreThanOnePodError) {
				logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", captureItem.SrcIP)
			} else {
				logrus.WithError(err).Debugf("Could not resolve %s to pod", captureItem.SrcIP)
			}
			continue
		}
		srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", srcPod.Name)
			continue
		}
		for _, dest := range captureItem.Destinations {
			if !strings.HasSuffix(dest, viper.GetString(config.ClusterDomainKey)) {
				// not a k8s service, ignore
				continue
			}
			ips, err := r.kubeFinder.ResolveServiceAddressToIps(ctx, dest)
			if err != nil {
				logrus.WithError(err).Warningf("Could not resolve service address %s", dest)
				continue
			}
			if len(ips) == 0 {
				logrus.Debugf("Service address %s is currently not backed by any pod, ignoring", dest)
				continue
			}
			destPod, err := r.kubeFinder.ResolveIpToPod(ctx, ips[0])
			if err != nil {
				if errors.Is(err, kubefinder.FoundMoreThanOnePodError) {
					logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", ips[0])
				} else {
					logrus.WithError(err).Debugf("Could not resolve %s to pod", ips[0])
				}
				continue
			}
			dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
			if err != nil {
				logrus.WithError(err).Debugf("Could not resolve pod %s to identity", destPod.Name)
				continue
			}
			r.intentsHolder.AddIntent(
				model.OtterizeServiceIdentity{Name: srcService, Namespace: srcPod.Namespace},
				model.OtterizeServiceIdentity{Name: dstService, Namespace: destPod.Namespace},
			)
		}
	}
	return true, nil
}

func (r *mutationResolver) ReportSocketScanResults(ctx context.Context, results model.SocketScanResults) (bool, error) {
	for _, socketScanItem := range results.Results {
		srcPod, err := r.kubeFinder.ResolveIpToPod(ctx, socketScanItem.SrcIP)
		if err != nil {
			if errors.Is(err, kubefinder.FoundMoreThanOnePodError) {
				logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", socketScanItem.SrcIP)
			} else {
				logrus.WithError(err).Debugf("Could not resolve %s to pod", socketScanItem.SrcIP)
			}
			continue
		}
		srcService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, srcPod)
		if err != nil {
			logrus.WithError(err).Debugf("Could not resolve pod %s to identity", srcPod.Name)
			continue
		}
		for _, destIp := range socketScanItem.DestIps {
			destPod, err := r.kubeFinder.ResolveIpToPod(ctx, destIp)
			if err != nil {
				if errors.Is(err, kubefinder.FoundMoreThanOnePodError) {
					logrus.WithError(err).Debugf("Ip %s belongs to more than one pod, ignoring", destIp)
				} else {
					logrus.WithError(err).Debugf("Could not resolve %s to pod", destIp)
				}
				continue
			}
			dstService, err := r.serviceIdResolver.ResolvePodToServiceIdentity(ctx, destPod)
			if err != nil {
				logrus.WithError(err).Debugf("Could not resolve pod %s to identity", destPod.Name)
				continue
			}
			r.intentsHolder.AddIntent(
				model.OtterizeServiceIdentity{Name: srcService, Namespace: srcPod.Namespace},
				model.OtterizeServiceIdentity{Name: dstService, Namespace: destPod.Namespace},
			)
		}
	}
	return true, nil
}

func (r *queryResolver) ServiceIntents(ctx context.Context, namespaces []string) ([]model.ServiceIntents, error) {
	result := make([]model.ServiceIntents, 0)
	for service, intents := range r.intentsHolder.GetIntentsPerService(namespaces) {
		result = append(result, model.ServiceIntents{Client: lo.ToPtr(service), Intents: intents})
	}
	// sorting by service name so results are more consistent
	sort.Slice(result, func(i, j int) bool {
		return result[i].Client.Name < result[j].Client.Name
	})
	return result, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
