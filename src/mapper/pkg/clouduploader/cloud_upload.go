package clouduploader

import (
	"context"
	"fmt"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"time"
)

type CloudUploader struct {
	intentsHolder       *resolvers.IntentsHolder
	config              Config
	tokenSrc            oauth2.TokenSource
	lastUploadTimestamp time.Time
	cloudClientFactory  cloudclient.FactoryFunction
}

func NewCloudUploader(intentsHolder *resolvers.IntentsHolder, config Config, cloudClientFactory cloudclient.FactoryFunction) *CloudUploader {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientId,
		ClientSecret: config.Secret,
		TokenURL:     fmt.Sprintf("%s/auth/tokens/token", config.apiAddress),
		AuthStyle:    oauth2.AuthStyleInParams,
	}

	tokenSrc := cfg.TokenSource(context.Background())

	return &CloudUploader{
		intentsHolder:      intentsHolder,
		config:             config,
		tokenSrc:           tokenSrc,
		cloudClientFactory: cloudClientFactory,
	}
}

func (c *CloudUploader) uploadDiscoveredIntents(ctx context.Context) {
	logrus.Info("Search for intents")

	client := c.cloudClientFactory(ctx, c.config.apiAddress, c.tokenSrc)

	lastUpdate := c.intentsHolder.LastIntentsUpdate()
	if !c.lastUploadTimestamp.Before(lastUpdate) {
		return
	}

	var intents []cloudclient.IntentInput
	for service, serviceIntents := range c.intentsHolder.GetIntentsPerService(nil) {
		for _, serviceIntent := range serviceIntents {
			var intent cloudclient.IntentInput
			intent.ClientName = lo.ToPtr(service.Name)
			intent.Namespace = lo.ToPtr(service.Namespace)
			intent.ServerName = lo.ToPtr(serviceIntent.Name)
			intent.ServerNamespace = lo.ToPtr(serviceIntent.Namespace)

			intents = append(intents, intent)
		}
	}

	if len(intents) == 0 {
		return
	}

	var discoveredIntents []*cloudclient.DiscoveredIntentInput
	for _, intent := range intents {
		input := cloudclient.DiscoveredIntentInput{
			Intent:       lo.ToPtr(intent),
			DiscoveredAt: lo.ToPtr(time.Now()),
		}

		discoveredIntents = append(discoveredIntents, lo.ToPtr(input))
	}

	uploadSuccess := client.ReportDiscoveredIntents(discoveredIntents)
	if uploadSuccess {
		c.lastUploadTimestamp = lastUpdate
	}
}

func (c *CloudUploader) reportStatus(ctx context.Context) {
	client := c.cloudClientFactory(ctx, c.config.apiAddress, c.tokenSrc)

	client.ReportComponentStatus(cloudclient.ComponentTypeNetworkMapper)
}

func (c *CloudUploader) PeriodicIntentsUpload(ctx context.Context) {
	cloudUploadTicker := time.NewTicker(time.Second * time.Duration(c.config.UploadInterval))

	logrus.Info("Starting cloud ticker")
	for {
		select {
		case <-cloudUploadTicker.C:
			c.uploadDiscoveredIntents(ctx)
			c.reportStatus(ctx)

		case <-ctx.Done():
			logrus.Info("Periodic upload exit")
			return
		}
	}
}
