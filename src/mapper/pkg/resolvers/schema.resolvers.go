package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"github.com/otterize/otternose/mapper/pkg/graph/generated"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
)

func (r *mutationResolver) ReportCaptureResults(ctx context.Context, results *model.CaptureResults) (*bool, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
