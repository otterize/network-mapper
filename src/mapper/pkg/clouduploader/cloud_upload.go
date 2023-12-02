package clouduploader

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
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

func (c *CloudUploader) NotifyIntents(ctx context.Context, intents []intentsstore.TimestampedIntent) {
	if len(intents) == 0 {
		return
	}

	discoveredIntents := lo.Map(intents, func(intent intentsstore.TimestampedIntent, _ int) *cloudclient.DiscoveredIntentInput {
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

	exponentialBackoff := backoff.NewExponentialBackOff()

	discoveredIntentsChunks := lo.Chunk(discoveredIntents, c.config.UploadBatchSize)
	currentChunk := 0
	err := backoff.Retry(func() error {
		for currentChunk < len(discoveredIntentsChunks) {
			err := c.client.ReportDiscoveredIntents(ctx, discoveredIntentsChunks[currentChunk])
			if err != nil {
				logrus.WithError(err).Errorf("Failed to report discovered intents chunk %d to cloud, retrying", currentChunk)
				return err
			}
			currentChunk += 1
		}
		return nil
	}, exponentialBackoff)
	if err != nil {
		logrus.WithError(err).Error("Failed to report discovered intents to cloud, giving up after 10 retries")
	}
}

func (c *CloudUploader) NotifyExternalTrafficIntents(ctx context.Context, intents []externaltrafficholder.TimestampedExternalTrafficIntent) {
	if len(intents) == 0 {
		return
	}

	logrus.Debugf("Got external traffic notification, len %d", len(intents))
	//logrus.Debugf("Saw external traffic, from '%s.%s' to '%s' (IP '%s')", srcSvcIdentity.Name, srcSvcIdentity.Namespace, dest.Destination, ip)

	discoveredIntents := lo.Map(intents, func(intent externaltrafficholder.TimestampedExternalTrafficIntent, _ int) cloudclient.ExternalTrafficDiscoveredIntentInput {
		output := cloudclient.ExternalTrafficDiscoveredIntentInput{
			DiscoveredAt: intent.Timestamp,
			Intent: cloudclient.ExternalTrafficIntentInput{
				ClientName: intent.Intent.Client.Name,
				Namespace:  intent.Intent.Client.Namespace,
				Target: cloudclient.DNSIPPairInput{
					DnsName: intent.Intent.DNSName,
				},
			},
		}
		for ip := range intent.Intent.IPs {
			output.Intent.Target.Ips = append(output.Intent.Target.Ips, string(ip))
		}
		return output
	})

	exponentialBackoff := backoff.NewExponentialBackOff()

	discoveredIntentsChunks := lo.Chunk(discoveredIntents, c.config.UploadBatchSize)
	currentChunk := 0
	err := backoff.Retry(func() error {
		for currentChunk < len(discoveredIntentsChunks) {
			err := c.client.ReportExternalTrafficDiscoveredIntents(ctx, discoveredIntentsChunks[currentChunk])
			if err != nil {
				logrus.WithError(err).Errorf("Failed to report discovered intents chunk %d to cloud, retrying", currentChunk)
				return err
			}
			currentChunk += 1
		}
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
