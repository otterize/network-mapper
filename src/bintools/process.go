package bintools

import (
	"debug/elf"
	"fmt"
	delve "github.com/go-delve/delve/pkg/goversion"
	"github.com/otterize/network-mapper/src/bintools/bininfo"
	"github.com/otterize/network-mapper/src/bintools/gobin"
	"log"
	"os"
)

func ProcessGoBinary(binPath string, functions []string) (res *gobin.GoBinaryInfo, err error) {
	f, err := os.Open(binPath)
	if err != nil {
		log.Fatalf("could not open file %s, %s", binPath, err)
	}
	defer func(f *os.File) {
		cErr := f.Close()
		if cErr == nil {
			err = cErr
		}
	}(f)

	// Parse the ELF file
	elfFile, err := elf.NewFile(f)
	if err != nil {
		log.Fatalf("file %s could not be parsed as an ELF file: %s", binPath, err)
	}

	// Determine the architecture of the binary
	arch, err := bininfo.GetArchitecture(elfFile)
	if err != nil {
		return nil, err
	}

	// Determine the Go version
	goVersion, rawVersion, err := gobin.FindGoVersion(f)
	if err != nil {
		return nil, err
	}

	// We only want to support Go versions 1.18 and above - where the ABI is register based
	if !goVersion.AfterOrEqual(delve.GoVersion{Major: 1, Minor: 18}) {
		return nil, fmt.Errorf("unsupported go version %s, only versions 1.18 and above are supported", rawVersion)
	}

	// Get symbols
	symbols, err := GetELFSymbolsByName(elfFile)
	if err != nil {
		return nil, fmt.Errorf("could not get symbols from ELF file: %w", err)
	}

	functionMetadata := make(map[string]gobin.FunctionMetadata, len(functions))

	for _, funcName := range functions {
		offset, err := SymbolToOffset(elfFile, symbols[funcName])
		if err != nil {
			return nil, fmt.Errorf("could not find location for function %q: %w", funcName, err)
		}

		// Find return locations
		var returnLocations []uint64
		symbol, ok := symbols[funcName]
		if !ok {
			return nil, fmt.Errorf("could not find function %q in symbols", funcName)
		}

		locations, err := FindReturnLocations(elfFile, arch, symbol, uint64(offset))
		if err != nil {
			return nil, fmt.Errorf("could not find return locations for function %q: %w", funcName, err)
		}

		returnLocations = locations

		functionMetadata[funcName] = gobin.FunctionMetadata{
			EntryAddress:    uint64(offset),
			ReturnAddresses: returnLocations,
		}
	}

	return &gobin.GoBinaryInfo{
		Arch:      arch,
		GoVersion: rawVersion,
		Functions: functionMetadata,
	}, nil
}
