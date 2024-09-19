package pcidata

import (
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"regexp"
)

func ContainsPaymentInformation(ctx ebpftypes.EventContext, data string) error {
	// Regular expression for possible credit card patterns (13-19 digits, allowing spaces, dashes, or dots as separators)
	re := regexp.MustCompile(cardRegex)

	// Find all matches
	matches := re.FindAllString(data, -1)

	// Filter matches based on valid credit card prefix and length
	for _, match := range matches {
		normalized := normalizeCardNumber(match)
		if isValidCardNumber(normalized) {
			// Set PCI tag
			ctx.Metadata.Tags[ebpftypes.EventTagPCI] = true
		}
	}

	return nil
}
