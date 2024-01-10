package mapperclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/intents-operator/src/shared/errors"
	"net/http"
)

type MapperClient interface {
	ReportIstioConnections(ctx context.Context, results IstioConnectionResults) error
	Health(ctx context.Context) error
}

type mapperClientImpl struct {
	mapperAddress string
	gqlClient     graphql.Client
}

func NewMapperClient(mapperAddress string) MapperClient {
	return &mapperClientImpl{
		mapperAddress: mapperAddress,
		gqlClient:     graphql.NewClient(mapperAddress, http.DefaultClient),
	}
}

func (c *mapperClientImpl) ReportIstioConnections(ctx context.Context, results IstioConnectionResults) error {
	_, err := reportIstioConnections(ctx, c.gqlClient, results)
	return errors.Wrap(err)
}

func (c *mapperClientImpl) Health(ctx context.Context) error {
	_, err := Health(ctx, c.gqlClient)
	return errors.Wrap(err)
}
