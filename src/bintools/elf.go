// Some functionality was copied from the Datadog-agent repository.
// Source: https://github.com/DataDog/datadog-agent
// Original license applies: Apache License Version 2.0.

package bintools

import (
	"debug/elf"
	"errors"
	"fmt"
	"github.com/otterize/network-mapper/src/bintools/gobin"
)

// GetELFSymbolsByName retrieves ELF symbols by name and returns them in a map.
func GetELFSymbolsByName(elfFile *elf.File) (map[string]elf.Symbol, error) {
	symbolMap := make(map[string]elf.Symbol)

	// Retrieve symbols from the .symtab section (static symbols)
	if symbols, err := elfFile.Symbols(); err == nil {
		for _, sym := range symbols {
			symbolMap[sym.Name] = sym
		}
	}

	// Retrieve symbols from the .dynsym section (dynamic symbols)
	if dynSymbols, err := elfFile.DynamicSymbols(); err == nil {
		for _, sym := range dynSymbols {
			symbolMap[sym.Name] = sym
		}
	}

	return symbolMap, nil
}

// SymbolToOffset returns the offset of the given symbol name in the given elf file.
func SymbolToOffset(f *elf.File, symbol elf.Symbol) (uint32, error) {
	if f == nil {
		return 0, errors.New("got nil elf file")
	}

	var sectionsToSearchForSymbol []*elf.Section

	for i := range f.Sections {
		if f.Sections[i].Flags == elf.SHF_ALLOC+elf.SHF_EXECINSTR {
			sectionsToSearchForSymbol = append(sectionsToSearchForSymbol, f.Sections[i])
		}
	}

	if len(sectionsToSearchForSymbol) == 0 {
		return 0, fmt.Errorf("symbol %q not found in file - no sections to search", symbol)
	}

	var executableSection *elf.Section

	// Find what section the symbol is in by checking the executable section's addr space.
	for m := range sectionsToSearchForSymbol {
		sectionStart := sectionsToSearchForSymbol[m].Addr
		sectionEnd := sectionStart + sectionsToSearchForSymbol[m].Size
		if symbol.Value >= sectionStart && symbol.Value < sectionEnd {
			executableSection = sectionsToSearchForSymbol[m]
			break
		}
	}

	if executableSection == nil {
		return 0, errors.New("could not find symbol in executable sections of binary")
	}

	// Calculate the Address of the symbol in the executable section
	return uint32(symbol.Value - executableSection.Addr + executableSection.Offset), nil
}

// FindReturnLocations returns the offsets of all the returns of the given func (sym) with the given offset.
func FindReturnLocations(elfFile *elf.File, arch gobin.GoArch, sym elf.Symbol, functionOffset uint64) ([]uint64, error) {
	textSection := elfFile.Section(".text")
	if textSection == nil {
		return nil, fmt.Errorf("no %q section found in binary file", ".text")
	}

	switch arch {
	case gobin.GoArchX86_64:
		return ScanFunction(textSection, sym, functionOffset, FindX86_64ReturnInstructions)
	case gobin.GoArchARM64:
		return ScanFunction(textSection, sym, functionOffset, FindARM64ReturnInstructions)
	default:
		return nil, gobin.ErrUnsupportedArch
	}
}
