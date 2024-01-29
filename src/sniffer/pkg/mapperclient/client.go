package mapperclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/intents-operator/src/shared/errors"
	"net/http"
)

type MapperClientImpl struct {
	mapperAddress string
	gqlClient     graphql.Client
}

func NewMapperClient(mapperAddress string) MapperClient {
	return &MapperClientImpl{
		mapperAddress: mapperAddress,
		gqlClient:     graphql.NewClient(mapperAddress, http.DefaultClient),
	}
}

func (c *MapperClientImpl) ReportCaptureResults(ctx context.Context, results CaptureResults) error {
	_, err := reportCaptureResults(ctx, c.gqlClient, results)
	return errors.Wrap(err)
}

func (c *MapperClientImpl) ReportSocketScanResults(ctx context.Context, results SocketScanResults) error {
	_, err := reportSocketScanResults(ctx, c.gqlClient, results)
	return errors.Wrap(err)
}

func (c *MapperClientImpl) Health(ctx context.Context) error {
	_, err := Health(ctx, c.gqlClient)
	return errors.Wrap(err)
}

type MapperClient interface {
	ReportCaptureResults(ctx context.Context, results CaptureResults) error
	ReportSocketScanResults(ctx context.Context, results SocketScanResults) error
	Health(ctx context.Context) error
}
