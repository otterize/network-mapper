package cloudclient

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type FactoryFunction func(ctx context.Context, apiAddress string, tokenSource oauth2.TokenSource) CloudClient

type CloudClient interface {
	ReportDiscoveredIntents(intents []*DiscoveredIntentInput) bool
	ReportComponentStatus(component ComponentType)
}

type CloudClientImpl struct {
	ctx    context.Context
	client graphql.Client
}

func NewClient(ctx context.Context, apiAddress string, tokenSource oauth2.TokenSource) CloudClient {
	url := fmt.Sprintf("%s/graphql/v1beta", apiAddress)
	client := graphql.NewClient(url, oauth2.NewClient(ctx, tokenSource))

	return &CloudClientImpl{
		client: client,
		ctx:    ctx,
	}
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
