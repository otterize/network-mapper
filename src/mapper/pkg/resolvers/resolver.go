package resolvers

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	kubeFinder                   *kubefinder.KubeFinder
	serviceIdResolver            *serviceidresolver.Resolver
	intentsHolder                *intentsstore.IntentsHolder
	externalTrafficIntentsHolder *externaltrafficholder.ExternalTrafficIntentsHolder
	awsIntentsHolder             *awsintentsholder.AWSIntentsHolder
	dnsCache                     *dnscache.DNSCache
}

func NewResolver(
	kubeFinder *kubefinder.KubeFinder,
	serviceIdResolver *serviceidresolver.Resolver,
	intentsHolder *intentsstore.IntentsHolder,
	externalTrafficHolder *externaltrafficholder.ExternalTrafficIntentsHolder,
	awsIntentsHolder *awsintentsholder.AWSIntentsHolder,
	dnsCache *dnscache.DNSCache,
) *Resolver {
	return &Resolver{
		kubeFinder:                   kubeFinder,
		serviceIdResolver:            serviceIdResolver,
		intentsHolder:                intentsHolder,
		externalTrafficIntentsHolder: externalTrafficHolder,
		awsIntentsHolder:             awsIntentsHolder,
		dnsCache:                     dnsCache,
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
