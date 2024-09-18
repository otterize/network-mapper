package types

import (
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
)

type EventContext struct {
	Data      []byte
	Event     otrzebpf.BpfSslEventT
	Container container.ContainerInfo
}
