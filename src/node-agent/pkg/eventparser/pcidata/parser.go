package pcidata

import (
	"fmt"
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/types"
	"github.com/sirupsen/logrus"
	"unicode"
)

type Parser struct {
	handlers []types.DataHandler[string]
}

// Parse parses the data to check if its plain text
func (p Parser) Parse(ctx ebpftypes.EventContext) (interface{}, error) {
	for _, b := range ctx.Data {
		// Check if the byte is a printable character or common whitespace (ASCII values 32-126 or newline/carriage return)
		if b > unicode.MaxASCII || (!unicode.IsPrint(rune(b)) && b != '\n' && b != '\r' && b != '\t') {
			return nil, fmt.Errorf("data is not plain text")
		}
	}
	logrus.Debugf("Got plain text data: %s\n", string(ctx.Data))

	return string(ctx.Data), nil
}

// RegisterHandler registers a handler for PCI data
func (p Parser) RegisterHandler(handler types.DataHandler[string]) {
	p.handlers = append(p.handlers, handler)
}

// RunHandlers executes all registered handlers on the parsed data
func (p Parser) RunHandlers(ctx ebpftypes.EventContext, data interface{}) error {
	str, ok := data.(string)
	if !ok {
		return fmt.Errorf("invalid type: expected string")
	}

	for _, handler := range p.handlers {
		if err := handler(ctx, str); err != nil {
			return err
		}
	}
	return nil
}
