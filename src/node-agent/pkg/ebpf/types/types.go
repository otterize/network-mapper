package types

import (
	"github.com/cilium/ebpf"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
)

// Direction signifies the traffic direction of a BPF event
type Direction int

const (
	DirectionEgress Direction = iota
	DirectionIngress
)

type BpfEventType string

const (
	BpfEventTypeUProbe    BpfEventType = "UProbe"
	BpfEventTypeURetProbe BpfEventType = "URetProbe"
)

type EventTag string

const (
	EventTagPCI EventTag = "PCI"
	EventTagPII EventTag = "PII"
)

type BpfProgram struct {
	Type        BpfEventType
	Symbol      string
	Handler     *ebpf.Program
	HandlerSpec *ebpf.ProgramSpec
	Address     uint64
}

// EventContext contains the data and metadata for a BPF event - used for parsing and handling of events
type EventContext struct {
	Data      []byte
	Event     otrzebpf.BpfSslEventT
	Container container.ContainerInfo
	Metadata  *EventMetadata // Metadata for the event - this is editable by parsers
}

// EventMetadata contains the parsed metadata for a BPF event
type EventMetadata struct {
	Tags map[EventTag]bool
}
