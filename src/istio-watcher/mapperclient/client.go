package mapperclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"net/http"
)

type MapperClient interface {
	ReportIstioConnections(ctx context.Context, results IstioConnectionResults) error
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
	return err
}
