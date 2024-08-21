package gobin

import (
	"debug/buildinfo"
	"debug/elf"
	"fmt"
	delve "github.com/go-delve/delve/pkg/goversion"
	"log"
	"os"
)

// FindGoVersion attempts to determine the Go version
func FindGoVersion(f *os.File) (gover delve.GoVersion, raw string, err error) {
	info, err := buildinfo.Read(f)
	if err != nil {
		log.Fatalf("error reading build info: %v", err)
	}

	versionString := info.GoVersion
	gover, ok := delve.Parse(versionString)
	if !ok {
		return gover, "", fmt.Errorf("error parsing go version %s", versionString)
	}

	return gover, versionString, nil
}

// GetGoArchitecture returns supported Go architecture for the given ELF file.
func GetGoArchitecture(elfFile *elf.File) (GoArch, error) {
	switch elfFile.FileHeader.Machine {
	case elf.EM_X86_64:
		return GoArchX86_64, nil
	case elf.EM_AARCH64:
		return GoArchARM64, nil
	}

	return "", ErrUnsupportedArch
}
