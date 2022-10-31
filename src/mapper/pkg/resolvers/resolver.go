package resolvers

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	kubeFinder        *kubefinder.KubeFinder
	serviceIdResolver *serviceidresolver.Resolver
	intentsHolder     *intentsHolder
}

func NewResolver(kubeFinder *kubefinder.KubeFinder, serviceIdResolver *serviceidresolver.Resolver) *Resolver {
	intentsHolder := NewIntentsHolder()
	storeFilePath := viper.GetString(config.StoreFileKey)
	if _, err := os.Stat(storeFilePath); err == nil {
		// Store file exists, we should load it
		err := intentsHolder.LoadStore(storeFilePath)
		if err != nil {
			logrus.WithError(err).Warning("Could not load store from previous run")
		} else {
			logrus.Info("Loaded data from previous run successfully")
		}
	}
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
