package httprequest

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/errors"
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/types"
	"github.com/sirupsen/logrus"
	"net/http"
)

type Parser struct {
	handlers []types.DataHandler[*http.Request]
}

// Parse parses the HTTP request from the given data
func (p *Parser) Parse(ctx ebpftypes.EventContext) (interface{}, error) {
	reader := bufio.NewReader(bytes.NewReader(ctx.Data))
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	req.RemoteAddr = ctx.Container.PodIP

	logrus.Debugf("Got HTTP request: %s", string(ctx.Data))
	return req, nil
}

// RegisterHandler registers a handler for HTTP events
func (p *Parser) RegisterHandler(handler types.DataHandler[*http.Request]) {
	p.handlers = append(p.handlers, handler)
}

// RunHandlers executes all registered handlers on the parsed data
func (p *Parser) RunHandlers(ctx ebpftypes.EventContext, data interface{}) error {
	req, ok := data.(*http.Request)
	if !ok {
		return fmt.Errorf("invalid type: expected *http.Request")
	}

	for _, handler := range p.handlers {
		if err := handler(ctx, req); err != nil {
			return err
		}
	}
	return nil
}
