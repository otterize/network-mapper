package textualdata

import (
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser/textualdata/helpers"
	"regexp"
	"slices"
	"strings"
)

func ContainsCreditCardData(ctx ebpftypes.EventContext, data string) error {
	// Regular expression for possible credit card patterns (13-19 digits, allowing spaces, dashes, or dots as separators)
	re := regexp.MustCompile(helpers.CardRegex)

	// Find all matches
	matches := re.FindAllString(data, -1)

	// Filter matches based on valid credit card prefix and length
	for _, match := range matches {
		normalized := helpers.NormalizeCardNumber(match)
		if helpers.IsValidCardNumber(normalized) {
			// Set PCI tag
			ctx.Metadata.AddTag(ebpftypes.EventTagPCI)
		}
	}

	return nil
}

func ContainsPaymentKeywords(ctx ebpftypes.EventContext, data string) error {
	keywords := slices.Concat(helpers.CardInfoKeywords, helpers.PaymentServiceKeywords, helpers.TransactionKeywords)

	// Check if any of the keywords are present in the text
	for _, keyword := range keywords {
		if strings.Contains(data, keyword) {
			// Set PII tag
			ctx.Metadata.AddTag(ebpftypes.EventTagPCI)
		}
	}

	return nil
}

func ContainsAddress(ctx ebpftypes.EventContext, data string) error {
	// Check if any of the keywords are present in the text
	for _, keyword := range helpers.AddressKeywords {
		if strings.Contains(data, keyword) {
			// Set PII tag
			ctx.Metadata.AddTag(ebpftypes.EventTagPII)
		}
	}

	return nil
}
