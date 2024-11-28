package clouduploader

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/awsintentsholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/externaltrafficholder"
	"github.com/otterize/network-mapper/src/mapper/pkg/incomingtrafficholder"
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
		toCloud := &cloudclient.DiscoveredIntentInput{
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
		if intent.Intent.Client.PodOwnerKind != nil && intent.Intent.Client.PodOwnerKind.Kind != "" {
			toCloud.Intent.ClientWorkloadKind = lo.ToPtr(intent.Intent.Client.PodOwnerKind.Kind)
		}
		if intent.Intent.Server.PodOwnerKind != nil && intent.Intent.Server.PodOwnerKind.Kind != "" {
			toCloud.Intent.ServerWorkloadKind = lo.ToPtr(intent.Intent.Server.PodOwnerKind.Kind)
		}
		if intent.Intent.Server.KubernetesService != nil {
			toCloud.Intent.ServerAlias = &cloudclient.ServerAliasInput{Name: intent.Intent.Server.KubernetesService, Kind: lo.ToPtr(serviceidentity.KindService)}
		}
		// debug log all the fields of intent input one by one with their values
		logrus.Debugf("intent ClientName: %s\t Namespace: %s\t ServerName: %s\t ServerNamespace: %s\t ClientWorkloadKind: %s\t ServerWorkloadKind: %s\t ServerAlias: %v", lo.FromPtr(toCloud.Intent.ClientName), lo.FromPtr(toCloud.Intent.Namespace), lo.FromPtr(toCloud.Intent.ServerName), lo.FromPtr(toCloud.Intent.ServerNamespace), lo.FromPtr(toCloud.Intent.ClientWorkloadKind), lo.FromPtr(toCloud.Intent.ServerWorkloadKind), lo.FromPtr(toCloud.Intent.ServerAlias))

		return toCloud
	})

	exponentialBackoff := backoff.NewExponentialBackOff()

	discoveredIntentsChunks := lo.Chunk(discoveredIntents, c.config.UploadBatchSize)
	currentChunk := 0
	err := backoff.Retry(func() error {
		for currentChunk < len(discoveredIntentsChunks) {
			err := c.client.ReportDiscoveredIntents(ctx, discoveredIntentsChunks[currentChunk])
			if err != nil {
				logrus.WithError(err).Errorf("Failed to report discovered intents chunk %d to cloud, retrying", currentChunk)
				return errors.Wrap(err)
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

	discoveredIntents := lo.Map(intents, func(intent externaltrafficholder.TimestampedExternalTrafficIntent, _ int) cloudclient.ExternalTrafficDiscoveredIntentInput {
		output := cloudclient.ExternalTrafficDiscoveredIntentInput{
			DiscoveredAt: intent.Timestamp,
			Intent: cloudclient.ExternalTrafficIntentInput{
				ClientName: intent.Intent.Client.Name,
				Namespace:  intent.Intent.Client.Namespace,
				Target: cloudclient.DNSIPPairInput{
					DnsName: lo.ToPtr(intent.Intent.DNSName),
				},
			},
		}
		for ip := range intent.Intent.IPs {
			output.Intent.Target.Ips = append(output.Intent.Target.Ips, lo.ToPtr(string(ip)))
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
				return errors.Wrap(err)
			}
			currentChunk += 1
		}
		return nil
	}, exponentialBackoff)
	if err != nil {
		logrus.WithError(err).Error("Failed to report discovered intents to cloud, giving up after 10 retries")
	}
}

func (c *CloudUploader) NotifyIncomingTrafficIntents(ctx context.Context, intents []incomingtrafficholder.TimestampedIncomingTrafficIntent) {
	if len(intents) == 0 {
		return
	}

	logrus.Debugf("Got incoming traffic notification, len %d", len(intents))

	discoveredIntents := lo.Map(intents, func(intent incomingtrafficholder.TimestampedIncomingTrafficIntent, _ int) cloudclient.IncomingTrafficDiscoveredIntentInput {
		output := cloudclient.IncomingTrafficDiscoveredIntentInput{
			DiscoveredAt: intent.Timestamp,
			Intent: cloudclient.IncomingTrafficIntentInput{
				ServerName: intent.Intent.Server.Name,
				Namespace:  intent.Intent.Server.Namespace,
				Source: cloudclient.IncomingInternetSourceInput{
					Ip: intent.Intent.IP,
				},
			},
		}
		return output
	})

	exponentialBackoff := backoff.NewExponentialBackOff()

	discoveredIntentsChunks := lo.Chunk(discoveredIntents, c.config.UploadBatchSize)
	currentChunk := 0
	err := backoff.Retry(func() error {
		for currentChunk < len(discoveredIntentsChunks) {
			err := c.client.ReportIncomingTrafficDiscoveredIntents(ctx, discoveredIntentsChunks[currentChunk])
			if err != nil {
				logrus.WithError(err).Errorf("Failed to report incoming traffic intents chunk %d to cloud, retrying", currentChunk)
				return errors.Wrap(err)
			}
			currentChunk += 1
		}
		return nil
	}, exponentialBackoff)
	if err != nil {
		logrus.WithError(err).Error("Failed to report incoming traffic intents to cloud, giving up after 10 retries")
	}
}

func (c *CloudUploader) NotifyAWSIntents(ctx context.Context, intents []awsintentsholder.AWSIntent) {
	if len(intents) == 0 {
		return
	}

	logrus.Debugf("Got AWS intents notification, len %d", len(intents))
	intentType := cloudclient.IntentTypeAws
	now := time.Now()

	err := c.client.ReportDiscoveredIntents(
		ctx,
		lo.Map(intents, func(intent awsintentsholder.AWSIntent, _ int) *cloudclient.DiscoveredIntentInput {
			toCloud := &cloudclient.DiscoveredIntentInput{
				DiscoveredAt: &now,
				Intent: &cloudclient.IntentInput{
					ClientName: &intent.Client.Name,
					Namespace:  &intent.Client.Namespace,
					ServerName: &intent.ARN,
					Type:       &intentType,
					AwsActions: lo.ToSlicePtr(intent.Actions),
				},
			}
			if intent.Client.PodOwnerKind != nil && intent.Client.PodOwnerKind.Kind != "" {
				toCloud.Intent.ClientWorkloadKind = lo.ToPtr(intent.Client.PodOwnerKind.Kind)
			}

			return toCloud
		}),
	)

	if err != nil {
		logrus.WithError(err).Error("Failed to report discovered intents to cloud")
	}
}

func (c *CloudUploader) NotifyAzureIntents(ctx context.Context, ops []model.AzureOperation) {
	if len(ops) == 0 {
		return
	}

	intentType := cloudclient.IntentTypeAzure
	now := time.Now()

	intents := lo.Map(ops, func(op model.AzureOperation, _ int) *cloudclient.DiscoveredIntentInput {
		toCloud := &cloudclient.DiscoveredIntentInput{
			DiscoveredAt: &now,
			Intent: &cloudclient.IntentInput{
				ClientName:       &op.ClientName,
				Namespace:        &op.ClientNamespace,
				ServerName:       &op.Scope,
				Type:             &intentType,
				AzureActions:     lo.ToSlicePtr(op.Actions),
				AzureDataActions: lo.ToSlicePtr(op.DataActions),
			},
		}
		return toCloud
	})

	err := c.client.ReportDiscoveredIntents(
		ctx,
		intents,
	)

	if err != nil {
		logrus.WithError(err).Error("Failed to report discovered intents to cloud")
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
