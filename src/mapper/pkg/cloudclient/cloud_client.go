package cloudclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/otterizecloud/otterizecloudclient"
	"github.com/sirupsen/logrus"
)

type CloudClient interface {
	ReportDiscoveredIntents(ctx context.Context, intents []*DiscoveredIntentInput) error
	ReportComponentStatus(ctx context.Context, component ComponentType) error
	ReportExternalTrafficDiscoveredIntents(ctx context.Context, intents []ExternalTrafficDiscoveredIntentInput) error
	ReportIncomingTrafficDiscoveredIntents(ctx context.Context, intents []IncomingTrafficDiscoveredIntentInput) error
	ReportK8sServices(ctx context.Context, namespace string, services []K8sServiceInput) error
	ReportK8sIngresses(ctx context.Context, namespace string, ingresses []K8sIngressInput) error
	ReportTrafficLevels(ctx context.Context, trafficLevels []TrafficLevelInput) error
	ReportNamespaceLabels(ctx context.Context, namespace string, labels []LabelInput) error
	ReportWorkloadsLabels(ctx context.Context, workloadsLabels []ReportServiceMetadataInput) error
	ReportK8sResourceEligibleForMetricsCollection(ctx context.Context, namespace string, reason EligibleForMetricsCollectionReason, resources []K8sResourceEligibleForMetricsCollectionInput) error
}

type CloudClientImpl struct {
	client graphql.Client
}

func NewClient(ctx context.Context) (*CloudClientImpl, bool, error) {
	client, ok, err := otterizecloudclient.NewClient(ctx)
	if !ok {
		return nil, false, nil
	} else if err != nil {
		return nil, true, errors.Wrap(err)
	}

	return &CloudClientImpl{client: client}, true, nil
}

func (c *CloudClientImpl) ReportDiscoveredIntents(ctx context.Context, intents []*DiscoveredIntentInput) error {
	logrus.Debug("Uploading intents to cloud, count: ", len(intents))
	_, err := ReportDiscoveredIntents(ctx, c.client, intents)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportExternalTrafficDiscoveredIntents(ctx context.Context, intents []ExternalTrafficDiscoveredIntentInput) error {
	logrus.Debug("Uploading external traffic intents to cloud, count: ", len(intents))

	_, err := ReportExternalTrafficDiscoveredIntents(ctx, c.client, intents)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportComponentStatus(ctx context.Context, component ComponentType) error {
	logrus.Debug("Uploading component status to cloud")

	_, err := ReportComponentStatus(ctx, c.client, component)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportIncomingTrafficDiscoveredIntents(ctx context.Context, intents []IncomingTrafficDiscoveredIntentInput) error {
	logrus.Debug("Uploading incoming traffic intents to cloud, count: ", len(intents))

	_, err := ReportIncomingTrafficDiscoveredIntents(ctx, c.client, intents)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportK8sServices(ctx context.Context, namespace string, services []K8sServiceInput) error {
	logrus.Debug("Uploading k8s services to cloud, count: ", len(services))

	_, err := ReportK8sServices(ctx, c.client, namespace, services)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportK8sIngresses(ctx context.Context, namespace string, ingresses []K8sIngressInput) error {
	logrus.Debug("Uploading k8s ingresses to cloud, count: ", len(ingresses))

	_, err := ReportK8sIngresses(ctx, c.client, namespace, ingresses)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportK8sResourceEligibleForMetricsCollection(ctx context.Context, namespace string, reason EligibleForMetricsCollectionReason, resources []K8sResourceEligibleForMetricsCollectionInput) error {
	logrus.Debug("Uploading k8s metrics collector resource to cloud, count: ", len(resources))

	_, err := ReportK8sResourceEligibleForMetricsCollection(ctx, c.client, namespace, reason, resources)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportTrafficLevels(
	ctx context.Context,
	trafficLevels []TrafficLevelInput,
) error {
	logrus.Debug("Uploading traffic info to cloud")

	_, err := ReportTrafficLevels(
		ctx,
		c.client,
		trafficLevels,
	)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportNamespaceLabels(ctx context.Context, namespace string, labels []LabelInput) error {
	logrus.Debugf("Uploading namespace %s labels to cloud", namespace)

	_, err := ReportNamespaceLabels(
		ctx,
		c.client,
		namespace,
		labels,
	)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (c *CloudClientImpl) ReportWorkloadsLabels(ctx context.Context, workloadsLabels []ReportServiceMetadataInput) error {
	logrus.Debugf("Uploading workloads labels to cloud")

	_, err := ReportWorkloadsLabels(
		ctx,
		c.client,
		workloadsLabels,
	)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}
