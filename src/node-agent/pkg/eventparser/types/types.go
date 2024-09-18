package types

import ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"

type DataHandler[T any] func(ctx ebpftypes.EventContext, data T) error

type Parser interface {
	Parse(ctx ebpftypes.EventContext) (interface{}, error)
	RunHandlers(ctx ebpftypes.EventContext, data interface{}) error
}
