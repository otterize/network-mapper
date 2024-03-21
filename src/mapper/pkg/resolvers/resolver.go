package resolvers

import (
	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"golang.org/x/sync/errgroup"
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
	dnsCaptureResults            chan model.CaptureResults
	socketScanResults            chan model.SocketScanResults
	kafkaMapperResults           chan model.KafkaMapperResults
	istioConnectionResults       chan model.IstioConnectionResults
	gotResultsCtx                context.Context
	gotResultsSignal             context.CancelFunc
}

func NewResolver(
	kubeFinder *kubefinder.KubeFinder,
	serviceIdResolver *serviceidresolver.Resolver,
	intentsHolder *intentsstore.IntentsHolder,
	externalTrafficHolder *externaltrafficholder.ExternalTrafficIntentsHolder,
	awsIntentsHolder *awsintentsholder.AWSIntentsHolder,
	dnsCache *dnscache.DNSCache,
) *Resolver {
	r := &Resolver{
		kubeFinder:                   kubeFinder,
		serviceIdResolver:            serviceIdResolver,
		intentsHolder:                intentsHolder,
		externalTrafficIntentsHolder: externalTrafficHolder,
		dnsCaptureResults:            make(chan model.CaptureResults, 200),
		socketScanResults:            make(chan model.SocketScanResults, 200),
		kafkaMapperResults:           make(chan model.KafkaMapperResults, 200),
		istioConnectionResults:       make(chan model.IstioConnectionResults, 200),
		awsIntentsHolder:             awsIntentsHolder,
		dnsCache:                     dnsCache,
	}
	r.gotResultsCtx, r.gotResultsSignal = context.WithCancel(context.Background())

	return r
}

func (r *Resolver) Register(e *echo.Echo) {
	c := generated.Config{Resolvers: r}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(c))
	e.Any("/query", func(c echo.Context) error {
		srv.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}

func (r *Resolver) RunForever(ctx context.Context) error {
	errgrp, errGrpCtx := errgroup.WithContext(ctx)
	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGrpCtx)
		return runHandleLoop(errGrpCtx, r.dnsCaptureResults, r.handleReportCaptureResults)
	})
	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGrpCtx)
		return runHandleLoop(errGrpCtx, r.socketScanResults, r.handleReportSocketScanResults)
	})
	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGrpCtx)
		return runHandleLoop(errGrpCtx, r.kafkaMapperResults, r.handleReportKafkaMapperResults)
	})
	errgrp.Go(func() error {
		defer bugsnag.AutoNotify(errGrpCtx)
		return runHandleLoop(errGrpCtx, r.istioConnectionResults, r.handleReportIstioConnectionResults)
	})
	err := errgrp.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
