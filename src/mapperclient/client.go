package mapperclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

type Client struct {
	client graphql.Client
}

func New(address string) *Client {
	// some usages of this lib pass /query, some don't
	if !strings.HasSuffix(address, "/query") {
		address = address + "/query"
	}

	logrus.Infof("Connecting to network-mapper at %s", address)

	return &Client{
		client: graphql.NewClient(address, http.DefaultClient),
	}
}

func (c *Client) ReportAWSOperation(ctx context.Context, operation []AWSOperation) error {
	_, err := reportAWSOperation(ctx, c.client, operation)
	return errors.Wrap(err)
}

func (c *Client) ReportGCPOperation(ctx context.Context, operation []GCPOperation) error {
	_, err := reportGCPOperation(ctx, c.client, operation)
	return errors.Wrap(err)
}

func (c *Client) ReportAzureOperation(ctx context.Context, operation []AzureOperation) error {
	_, err := reportAzureOperation(ctx, c.client, operation)
	return errors.Wrap(err)
}

func (c *Client) ReportKafkaMapperResults(ctx context.Context, results KafkaMapperResults) error {
	_, err := reportKafkaMapperResults(ctx, c.client, results)
	return errors.Wrap(err)
}

func (c *Client) ReportCaptureResults(ctx context.Context, results CaptureResults) error {
	_, err := reportCaptureResults(ctx, c.client, results)
	return errors.Wrap(err)
}

func (c *Client) ReportTCPCaptureResults(ctx context.Context, results CaptureTCPResults) error {
	_, err := reportTCPCaptureResults(ctx, c.client, results)
	return err
}

func (c *Client) ReportSocketScanResults(ctx context.Context, results SocketScanResults) error {
	_, err := reportSocketScanResults(ctx, c.client, results)
	return errors.Wrap(err)
}

func (c *Client) ReportTrafficLevels(ctx context.Context, results TrafficLevelResults) error {
	_, err := reportTrafficLevelResults(ctx, c.client, results)
	return errors.Wrap(err)
}

func (c *Client) Health(ctx context.Context) error {
	_, err := Health(ctx, c.client)
	return errors.Wrap(err)
}
