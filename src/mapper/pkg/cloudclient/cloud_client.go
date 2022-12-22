package cloudclient

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type FactoryFunction func(ctx context.Context, apiAddress string, tokenSource oauth2.TokenSource) CloudClient

type CloudClient interface {
	ReportDiscoveredSourcedIntents(namespace string, intents []IntentInput) bool
}

type CloudClientImpl struct {
	ctx    context.Context
	client graphql.Client
}

func NewClient(ctx context.Context, apiAddress string, tokenSource oauth2.TokenSource) CloudClient {
	uri := viper.GetString(config.CloudGraphQLEndpointKey)
	url := fmt.Sprintf("%s/%s", apiAddress, uri)
	client := graphql.NewClient(url, oauth2.NewClient(ctx, tokenSource))

	return &CloudClientImpl{
		client: client,
		ctx:    ctx,
	}
}

func (c *CloudClientImpl) ReportDiscoveredSourcedIntents(namespace string, intents []IntentInput) bool {
	logrus.Info("Uploading intents to cloud, count: ", len(intents))
	_, err := ReportDiscoveredSourcedIntents(c.ctx, c.client, namespace, intents)
	if err != nil {
		logrus.Error("Failed to upload intents to cloud ", err)
		return false
	}

	return true
}
