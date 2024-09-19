package pcidata

import (
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/stretchr/testify/suite"
	"testing"
)

type PciParserTestSuite struct {
	suite.Suite
}

func (s *PciParserTestSuite) SetupSuite() {
}

func (s *PciParserTestSuite) TestParse() {
	parser := Parser{}
	validEvent := ebpftypes.EventContext{Data: []byte("Hello, World!\n")}
	invalidEvent := ebpftypes.EventContext{Data: []byte{0x01, 0x02, 0x03, 0xFF}}

	data, err := parser.Parse(validEvent)
	s.Require().NoError(err)
	_, ok := data.(string)
	s.Require().True(ok)

	data, err = parser.Parse(invalidEvent)
	s.Require().Error(err)
}

func TestPciParserTestSuite(t *testing.T) {
	suite.Run(t, new(PciParserTestSuite))
}
