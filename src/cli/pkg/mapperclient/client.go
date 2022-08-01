package mapperclient

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/otterize/otternose/cli/pkg/config"
	"github.com/otterize/otternose/cli/pkg/portforwarder"
	"github.com/spf13/viper"
	"net/http"
)

type Client struct {
	address string
	client  graphql.Client
}

func NewClient(address string) *Client {
	return &Client{
		address: address,
		client:  graphql.NewClient(address+"/query", http.DefaultClient),
	}
}

func WithClient(f func(c *Client) error) error {
	portFwdCtx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	portForwarder := portforwarder.NewPortForwarder(viper.GetString(config.MapperNamespaceKey), viper.GetString(config.MapperServiceNameKey), viper.GetInt(config.MapperServicePortKey))
	localPort, err := portForwarder.Start(portFwdCtx)
	if err != nil {
		return err
	}
	c := NewClient(fmt.Sprintf("http://localhost:%d", localPort))
	return f(c)
}

func (c *Client) FormattedCRDs(ctx context.Context) (string, error) {
	res, err := FormattedCRDs(ctx, c.client)
	if err != nil {
		return "", err
	}
	return res.FormattedCRDs, nil
}

func (c *Client) ServiceIntents(ctx context.Context) ([]ServiceIntentsServiceIntents, error) {
	res, err := ServiceIntents(ctx, c.client)
	if err != nil {
		return nil, err
	}
	return res.ServiceIntents, nil
}
