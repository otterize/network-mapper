package clouduploader

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/resolvers"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"time"
)

type CloudUploader struct {
	intentsHolder *resolvers.IntentsHolder
	config        Config
	client        cloudclient.CloudClient
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

	var discoveredIntents []*cloudclient.DiscoveredIntentInput
	for _, intent := range c.intentsHolder.GetNewIntentsSinceLastGet() {
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

	exponentialBackoff := backoff.NewExponentialBackOff()

	err := backoff.Retry(func() error {
		err := c.client.ReportDiscoveredIntents(ctx, discoveredIntents)
		if err != nil {
			logrus.WithError(err).Error("Failed to report discovered intents to cloud, retrying")
		}
		return err
	}, backoff.WithMaxRetries(exponentialBackoff, 10))
	if err != nil {
		logrus.WithError(err).Error("Failed to report discovered intents to cloud, giving up after 10 retries")
	}
}

func (c *CloudUploader) reportStatus(ctx context.Context) {
	err := c.client.ReportComponentStatus(ctx, cloudclient.ComponentTypeNetworkMapper)
	if err != nil {
		logrus.WithError(err).Error("Failed to report component status to cloud")
	}
}

func (c *CloudUploader) PeriodicIntentsUpload(ctx context.Context) {
	logrus.Info("Starting periodic intents upload")

	for {
		select {
		case <-time.After(c.config.UploadInterval):
			c.uploadDiscoveredIntents(ctx)

		case <-ctx.Done():
			return
		}
	}
}

func (c *CloudUploader) PeriodicStatusReport(ctx context.Context) {
	logrus.Info("Starting status reporting")
	c.reportStatus(ctx)

	for {
		select {
		case <-time.After(c.config.UploadInterval):
			c.reportStatus(ctx)

		case <-ctx.Done():
			return
		}
	}
}
