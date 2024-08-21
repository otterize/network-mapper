package gobin

import (
	"errors"
)

// ErrUnsupportedArch is returned when an architecture given as a parameter is not supported.
var ErrUnsupportedArch = errors.New("got unsupported arch")

// GoArch only includes go architectures that we support in the ebpf code.
type GoArch string

const (
	GoArchX86_64 GoArch = "amd64"
	GoArchARM64  GoArch = "arm64"
)

// FunctionMetadata  used to attach a uprobe to a function.
type FunctionMetadata struct {
	EntryAddress    uint64
	ReturnAddresses []uint64
}

type GoBinaryInfo struct {
	Arch      GoArch
	GoVersion string
	Functions map[string]FunctionMetadata
}
