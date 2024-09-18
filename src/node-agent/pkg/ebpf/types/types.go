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

type BpfProgram struct {
	Type        BpfEventType
	Symbol      string
	Handler     *ebpf.Program
	HandlerSpec *ebpf.ProgramSpec
	Address     uint64
}

type EventContext struct {
	Data      []byte
	Event     otrzebpf.BpfSslEventT
	Container container.ContainerInfo
}
