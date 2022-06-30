package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/otterize/otternose/mapper/pkg/reconcilers"
	"github.com/otterize/otternose/mapper/pkg/resolvers"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	// start manager with operators
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{MetricsBindAddress: "0"})
	if err != nil {
		logrus.Errorf("unable to set up overall controller manager: %s", err)
		os.Exit(1)
	}
	podsReconciler := reconcilers.NewPodsReconciler(mgr.GetClient())
	err = podsReconciler.Register(mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	endpointsReconciler := reconcilers.NewEndpointsReconciler(mgr.GetClient())
	err = endpointsReconciler.Register(mgr)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	go func() {
		if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			logrus.Error(err, "unable to run manager")
			os.Exit(1)

		}
	}()

	// start API server
	e := echo.New()
	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Use(middleware.RemoveTrailingSlash())
	resolver := resolvers.NewResolver(podsReconciler, endpointsReconciler)
	resolver.Register(e)

	err = e.Start("0.0.0.0:9090")
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

}
