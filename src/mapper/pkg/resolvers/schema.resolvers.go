package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/mapper/pkg/config"
	"github.com/spf13/viper"
	"strings"

	"github.com/otterize/otternose/mapper/pkg/graph/generated"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
	"github.com/sirupsen/logrus"
)

func (r *mutationResolver) ReportCaptureResults(ctx context.Context, results model.CaptureResults) (*bool, error) {
	for _, result := range results.Results {
		pod, ok := r.podsReconciler.ResolveIpToPod(result.SrcIP)
		if !ok {
			logrus.Warningf("Ip %s didn't match any pod", result.SrcIP)
			continue
		}
		destinationsAsPods := make([]string, 0)
		for _, dest := range result.Destinations {
			if !strings.HasSuffix(dest, viper.GetString(config.ClusterDomainKey)) {
				// not a k8s service, ignore
				continue
			}
			ips, ok := r.endpointsReconciler.ResolveServiceAddressToIps(dest)
			if !ok {
				logrus.Warningf("Could not resolve service address %s", dest)
				continue
			}
			destPod, ok := r.podsReconciler.ResolveIpToPod(ips[0])
			if !ok {
				logrus.Warningf("Could not resolve pod IP %s", ips[0])
				continue
			}
			destinationsAsPods = append(destinationsAsPods, fmt.Sprintf("%s.%s", destPod.Name, destPod.Namespace))
		}
		fmt.Printf("%s %s.%s: %v\n", result.SrcIP, pod.Name, pod.Namespace, destinationsAsPods)
	}
	return nil, nil
}

func (r *queryResolver) Placeholder(ctx context.Context) (*bool, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
