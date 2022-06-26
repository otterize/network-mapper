package client

import (
	"context"
	"github.com/Khan/genqlient/graphql"
	"net/http"
)

type MapperClient struct {
	mapperAddress string
	gqlClient     graphql.Client
}

func NewMapperClient(mapperAddress string) *MapperClient {
	return &MapperClient{
		mapperAddress: mapperAddress,
		gqlClient:     graphql.NewClient(mapperAddress, http.DefaultClient),
	}
}

func (c *MapperClient) ReportCaptureResults(ctx context.Context, results CaptureResults) error {
	_, err := ReportCaptureResults(ctx, c.gqlClient, results)
	return err
}
