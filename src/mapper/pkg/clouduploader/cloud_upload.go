package clouduploader

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"time"
)

type CloudUploader struct {
	intentsHolder       *resolvers.IntentsHolder
	config              Config
	lastUploadTimestamp time.Time
	client              cloudclient.CloudClient
}

func NewCloudUploader(intentsHolder *resolvers.IntentsHolder, config Config, client cloudclient.CloudClient) *CloudUploader {
	return &CloudUploader{
		intentsHolder: intentsHolder,
		config:        config,
		client:        client,
	}
}

func (c *CloudUploader) uploadDiscoveredIntents(ctx context.Context) {
	logrus.Info("Search for intents")

	lastUpdate := c.intentsHolder.LastIntentsUpdate()
	if !c.lastUploadTimestamp.Before(lastUpdate) {
		return
	}

	var discoveredIntents []*cloudclient.DiscoveredIntentInput
	for _, intent := range c.intentsHolder.GetIntents(nil) {
		var discoveredIntent cloudclient.IntentInput
		discoveredIntent.ClientName = lo.ToPtr(intent.Source.Name)
		discoveredIntent.Namespace = lo.ToPtr(intent.Source.Namespace)
		discoveredIntent.ServerName = lo.ToPtr(intent.Destination.Name)
		discoveredIntent.ServerNamespace = lo.ToPtr(intent.Destination.Namespace)

		input := &cloudclient.DiscoveredIntentInput{
			DiscoveredAt: lo.ToPtr(intent.Timestamp),
			Intent:       &discoveredIntent,
		}

		discoveredIntents = append(discoveredIntents, input)
	}

	if len(discoveredIntents) == 0 {
		return
	}

	uploadSuccess := c.client.ReportDiscoveredIntents(discoveredIntents)
	if uploadSuccess {
		c.lastUploadTimestamp = lastUpdate
	}
}

func (c *CloudUploader) reportStatus(ctx context.Context) {
	c.client.ReportComponentStatus(cloudclient.ComponentTypeNetworkMapper)
}

func (c *CloudUploader) PeriodicIntentsUpload(ctx context.Context) {
	cloudUploadTicker := time.NewTicker(time.Second * time.Duration(c.config.UploadInterval))

	logrus.Info("Starting cloud ticker")
	c.reportStatus(ctx)

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
