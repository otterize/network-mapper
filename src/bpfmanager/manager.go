package bpfmanager

import (
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"log"
)

type BpfEventType string

const (
	BpfEventTypeUProbe    BpfEventType = "UProbe"
	BpfEventTypeURetProbe BpfEventType = "URetProbe"
)

type BpfProgram struct {
	Type    BpfEventType
	Symbol  string
	Address uint64
	Handler *ebpf.Program
}

type ProbeManager struct {
	binPath     string
	programs    []BpfProgram
	activeLinks []link.Link
}

func (pm *ProbeManager) RegisterProgram(program BpfProgram) {
	pm.programs = append(pm.programs, program)
}

func (pm *ProbeManager) Init() error {
	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	ex, err := link.OpenExecutable(pm.binPath)
	if err != nil {
		return fmt.Errorf("error opening executable: %s", err)
	}

	for _, program := range pm.programs {
		var up link.Link

		opts := &link.UprobeOptions{}
		if program.Address != 0 {
			opts.Address = program.Address
		}

		log.Printf("Registering program: %s on address: %d", program.Symbol, program.Address)

		switch program.Type {
		case BpfEventTypeUProbe:
			up, err = ex.Uprobe(program.Symbol, program.Handler, opts)
		case BpfEventTypeURetProbe:
			up, err = ex.Uretprobe(program.Symbol, program.Handler, opts)
		}
		if err != nil {
			return fmt.Errorf("error creating uretprobe: %s", err)
		}

		pm.activeLinks = append(pm.activeLinks, up)
	}

	return nil
}

func (pm *ProbeManager) Close() {
	for _, activeLink := range pm.activeLinks {
		err := activeLink.Close()
		if err != nil {
			fmt.Printf("error closing link: %v\n", err)
		}
	}
}

func NewProbeManager(binPath string) *ProbeManager {
	return &ProbeManager{
		binPath: binPath,
	}
}
