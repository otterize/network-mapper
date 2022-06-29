package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"

	"github.com/otterize/otternose/mapper/pkg/graph/generated"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
)

func (r *mutationResolver) ReportCaptureResults(_ context.Context, results model.CaptureResults) (*bool, error) {
	for _, result := range results.Results {
		podInfo, ok := r.podsOperator.ResolveIpToPodInfo(result.SrcIP)
		if !ok {
			logrus.Warningf("Ip %s didn't match any pod", result.SrcIP)
		}
		fmt.Printf("%s %s.%s: %v\n", result.SrcIP, podInfo.Namespace, podInfo.Name, result.Destinations)
	}
	return nil, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
