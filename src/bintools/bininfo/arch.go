package bininfo

import (
	"debug/elf"
	"errors"
)

// ErrUnsupportedArch is returned when an architecture given as a parameter is not supported.
var ErrUnsupportedArch = errors.New("got unsupported arch")

// Arch only includes go architectures that we support in the ebpf code.
type Arch string

const (
	ArchX86_64 Arch = "amd64"
	ArchARM64  Arch = "arm64"
)

// GetArchitecture returns supported Go architecture for the given ELF file.
func GetArchitecture(elfFile *elf.File) (Arch, error) {
	switch elfFile.FileHeader.Machine {
	case elf.EM_X86_64:
		return ArchX86_64, nil
	case elf.EM_AARCH64:
		return ArchARM64, nil
	}

	return "", ErrUnsupportedArch
}
