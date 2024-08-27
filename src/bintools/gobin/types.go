package gobin

import (
	"github.com/otterize/network-mapper/src/bintools/bininfo"
)

// FunctionMetadata  used to attach a uprobe to a function.
type FunctionMetadata struct {
	EntryAddress    uint64
	ReturnAddresses []uint64
}

type GoBinaryInfo struct {
	Arch      bininfo.Arch
	GoVersion string
	Functions map[string]FunctionMetadata
}
