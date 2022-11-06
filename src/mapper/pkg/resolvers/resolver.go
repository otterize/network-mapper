package resolvers

import (
	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/sirupsen/logrus"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	kubeFinder        *kubefinder.KubeFinder
	serviceIdResolver *serviceidresolver.Resolver
	intentsHolder     *intentsHolder
}

func NewResolver(kubeFinder *kubefinder.KubeFinder, serviceIdResolver *serviceidresolver.Resolver, intentsHolder *intentsHolder) *Resolver {
	return &Resolver{
		kubeFinder:        kubeFinder,
		serviceIdResolver: serviceIdResolver,
		intentsHolder:     intentsHolder,
	}
}

func (r *Resolver) Register(e *echo.Echo) {
	c := generated.Config{Resolvers: r}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(c))
	e.Any("/query", func(c echo.Context) error {
		srv.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}

func (r *Resolver) LoadStore(ctx context.Context) {
	err := r.intentsHolder.LoadStore(ctx)
	if err != nil {
		logrus.WithError(err).Warning("Failed to load state from previous runs")
	}
}
