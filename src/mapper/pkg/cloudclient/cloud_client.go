package cloudclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/intents-operator/src/shared/otterizecloud/otterizecloudclient"
	"github.com/sirupsen/logrus"
)

type CloudClient interface {
	ReportDiscoveredIntents(intents []*DiscoveredIntentInput) bool
	ReportComponentStatus(component ComponentType)
}

type CloudClientImpl struct {
	ctx    context.Context
	client graphql.Client
}

func NewClient(ctx context.Context) (CloudClient, bool, error) {
	client, ok, err := otterizecloudclient.NewClient(ctx)
	if !ok {
		return nil, false, nil
	} else if err != nil {
		return nil, true, err
	}

	return &CloudClientImpl{client: client}, true, nil
}

func (c *CloudClientImpl) ReportDiscoveredIntents(intents []*DiscoveredIntentInput) bool {
	logrus.Info("Uploading intents to cloud, count: ", len(intents))

	_, err := ReportDiscoveredIntents(c.ctx, c.client, intents)
	if err != nil {
		logrus.Error("Failed to upload intents to cloud ", err)
		return false
	}

	return true
}

func (c *CloudClientImpl) ReportComponentStatus(component ComponentType) {
	logrus.Info("Uploading component to cloud")

	_, err := ReportComponentStatus(c.ctx, c.client, component)
	if err != nil {
		logrus.Error("Failed to upload component to cloud ", err)
	}
}
