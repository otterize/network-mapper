package pcidata

import (
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/stretchr/testify/suite"
	"testing"
)

// PCIHTTPMessages contains example HTTP messages that contain PCI data (urlencoded, json, and multipart/form-data)
var PCIHTTPMessages = []string{
	"POST /payment/submit HTTP/1.1\nHost: payments.example.com\nContent-Type: application/x-www-form-urlencoded\nContent-Length: 186\n\nname=John+Doe&email=johndoe%40example.com&address=123+Main+St&city=New+York&state=NY&zip=10001&card_number=4111111111111111&expiration_date=12%2F25&cvv=123&amount=49.99&currency=USD\n",
	"POST /api/v1/checkout HTTP/1.1\nHost: payments.example.com\nContent-Type: application/json\nContent-Length: 235\n\n{\n  \"name\": \"John Doe\",\n  \"email\": \"johndoe@example.com\",\n  \"billing_address\": {\n    \"address\": \"123 Main St\",\n    \"city\": \"New York\",\n    \"state\": \"NY\",\n    \"zip\": \"10001\"\n  },\n  \"payment_info\": {\n    \"card_number\": \"4111 1111 1111 1111\",\n    \"expiration_date\": \"12/25\",\n    \"cvv\": \"123\"\n  },\n  \"amount\": 49.99,\n  \"currency\": \"USD\"\n}\n",
	"POST /submit-payment HTTP/1.1\nHost: payments.example.com\nContent-Type: multipart/form-data; boundary=---1234\nContent-Length: 421\n\n---1234\nContent-Disposition: form-data; name=\"card_number\"\n\n4111-1111-1111-1111\n---1234\nContent-Disposition: form-data; name=\"expiry_date\"\n\n12/25\n---1234\nContent-Disposition: form-data; name=\"cvv\"\n\n123\n---1234\nContent-Disposition: form-data; name=\"cardholder_name\"\n\nJohn Doe\n---1234\nContent-Disposition: form-data; name=\"billing_address\"\n\n123 Main St, City, Country\n---1234--",
}

type PciHandlersTestSuite struct {
	suite.Suite
}

func (s *PciHandlersTestSuite) SetupSuite() {
}

func (s *PciHandlersTestSuite) TestContainsPaymentInformation() {
	for _, msg := range PCIHTTPMessages {
		ctx := ebpftypes.EventContext{
			Data:     []byte(msg),
			Metadata: &ebpftypes.EventMetadata{},
		}
		err := ContainsPaymentInformation(ctx, string(ctx.Data))
		s.NoError(err)
		s.True(ctx.Metadata.Tags[ebpftypes.EventTagPCI])
	}
}

func TestPciHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(PciHandlersTestSuite))
}
