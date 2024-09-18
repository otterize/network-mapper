package eventparser

import (
	"github.com/otterize/intents-operator/src/shared/errors"
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/httprequest"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/httpresponse"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/types"
	"github.com/sirupsen/logrus"
)

// Contains all the parsers that can be used to parse events
var parsers = make(map[string]types.Parser)

func init() {
	// Initialize HTTP request parser
	httpRequestParser := &httprequest.Parser{}
	httpRequestParser.RegisterHandler(httprequest.HandleAwsRequest)
	parsers["httprequest"] = httpRequestParser

	// Initialize HTTP response parser
	httpResponseParser := &httpresponse.Parser{}
	parsers["httpresponse"] = httpResponseParser
}

func ProcessEvent(ctx ebpftypes.EventContext) error {
	for protocol, parser := range parsers {
		parsedData, err := parser.Parse(ctx)
		if err != nil {
			logrus.Debugf("Event %d cannot be parsed using %s parser: %v\n", ctx.Event.Meta.Pid, protocol, err)
			continue
		}

		// Run handlers for this parser
		err = parser.RunHandlers(ctx, parsedData)
		if err != nil {
			logrus.Debugf("Handler error for protocol %s: %v\n", protocol, err)
			return errors.Wrap(err)
		}
	}

	return nil
}
