package ebpf

import (
	"github.com/cilium/ebpf"
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

type ProbeKey struct {
	inode       uint64
	fnName      string
	address     uint64
	retprobe    bool
	programName string
}

type BpfProgram struct {
	Type        BpfEventType
	Symbol      string
	Handler     *ebpf.Program
	HandlerSpec *ebpf.ProgramSpec
	Address     uint64
}

func (p *BpfProgram) GetUniqueKey(inode uint64) ProbeKey {
	return ProbeKey{
		address:  p.Address,
		inode:    inode,
		fnName:   p.Symbol,
		retprobe: p.Type == BpfEventTypeURetProbe,
	}
}
