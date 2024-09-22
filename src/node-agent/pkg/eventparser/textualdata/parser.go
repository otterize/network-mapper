package textualdata

import (
	"fmt"
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/types"
	"github.com/sirupsen/logrus"
	"strings"
	"unicode"
)

type Parser struct {
	handlers []types.DataHandler[string]
}

// Parse parses the data to check if its plain text - allow up to 30% of the data to be non-printable
func (p Parser) Parse(ctx ebpftypes.EventContext) (interface{}, error) {
	limit := 0.7 // Default to 70% if threshold is out of range

	totalBytes := len(ctx.Data)
	if totalBytes == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	plainTextBytes := 0
	for _, b := range ctx.Data {
		// Check if the byte is a printable character or common whitespace (ASCII values 32-126 or newline/carriage return)
		if b <= unicode.MaxASCII || (unicode.IsPrint(rune(b)) || b == '\n' || b == '\r' || b == '\t' || b == '\f') {
			plainTextBytes += 1
		}
	}

	// Calculate the percentage of plain text bytes
	percentage := float64(plainTextBytes) / float64(totalBytes)
	if percentage < limit {
		return nil, fmt.Errorf("most data is not plain text %f", percentage)
	}

	logrus.Debugf("got plain text data [%f]: %s\n", percentage, string(ctx.Data))
	return strings.ToLower(string(ctx.Data)), nil
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
