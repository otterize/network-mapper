package clouduploader

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"time"
)

type CloudUploader struct {
	intentsHolder *intentsstore.IntentsHolder
	config        Config
	client        cloudclient.CloudClient
}

func NewCloudUploader(intentsHolder *intentsstore.IntentsHolder, config Config, client cloudclient.CloudClient) *CloudUploader {
	return &CloudUploader{
		intentsHolder: intentsHolder,
		config:        config,
		client:        client,
	}
}

func (c *CloudUploader) uploadDiscoveredIntents(ctx context.Context) {
	logrus.Info("Search for intents")

	discoveredIntents := lo.Map(c.intentsHolder.GetNewIntentsSinceLastGet(), func(intent intentsstore.TimestampedIntent, _ int) *cloudclient.DiscoveredIntentInput {
		return &cloudclient.DiscoveredIntentInput{
			DiscoveredAt: lo.ToPtr(intent.Timestamp),
			Intent: &cloudclient.IntentInput{
				ClientName:      &intent.Intent.Client.Name,
				Namespace:       &intent.Intent.Client.Namespace,
				ServerName:      &intent.Intent.Server.Name,
				ServerNamespace: &intent.Intent.Server.Namespace,
				Type:            modelIntentTypeToAPI(intent.Intent.Type),
				Topics: lo.Map(intent.Intent.KafkaTopics,
					func(item model.KafkaConfig, _ int) *cloudclient.KafkaConfigInput {
						return lo.ToPtr(modelKafkaConfToAPI(item))
					},
				),
				Resources: httpResourceToHTTPConfInput(intent.Intent.HTTPResources),
			},
		}
	})

	if len(discoveredIntents) == 0 {
		return
	}

	exponentialBackoff := backoff.NewExponentialBackOff()

	discoveredIntentsChunks := lo.Chunk(discoveredIntents, c.config.UploadBatchSize)
	currentChunk := 0
	err := backoff.Retry(func() error {
		err := c.client.ReportDiscoveredIntents(ctx, discoveredIntentsChunks[currentChunk])
		if err != nil {
			logrus.WithError(err).Error("Failed to report discovered intents to cloud, retrying")
			return err
		}
		currentChunk += 1
		return nil
	}, exponentialBackoff)
	if err != nil {
		logrus.WithError(err).Error("Failed to report discovered intents to cloud, giving up after 10 retries")
	}
}

func httpResourceToHTTPConfInput(resources []model.HTTPResource) []*cloudclient.HTTPConfigInput {
	httpGQLInputs := make([]*cloudclient.HTTPConfigInput, 0)
	for _, resource := range resources {
		httpGQLInputs = append(httpGQLInputs, &cloudclient.HTTPConfigInput{
			Path: lo.ToPtr(resource.Path),
			Methods: lo.Map(resource.Methods, func(method model.HTTPMethod, _ int) *cloudclient.HTTPMethod {
				return lo.ToPtr(modelHTTPMethodToAPI(method))
			})})
	}
	return httpGQLInputs
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
