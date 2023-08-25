package mapperclient

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"net/http"
)

type MapperClient interface {
	ReportKafkaMapperResults(ctx context.Context, results KafkaMapperResults) error
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

func (c *mapperClientImpl) ReportKafkaMapperResults(ctx context.Context, results KafkaMapperResults) error {
	_, err := reportKafkaMapperResults(ctx, c.gqlClient, results)
	return err
}
