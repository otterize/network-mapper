package ebpf

import (
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/pkg/errors"
)

type Tracer struct {
	targetMap *ebpf.Map
	probes    map[ProbeKey]link.Link
	reader    *EventReader
}

func NewTracer(reader *EventReader) *Tracer {
	return &Tracer{
		targetMap: otrzebpf.Objs.Targets,
		probes:    make(map[ProbeKey]link.Link),
		reader:    reader,
	}
}

func (t *Tracer) attachBpfProgram(ex *link.Executable, binaryInode uint64, program BpfProgram) (err error) {
	key := getProbeKey(program, binaryInode)
	if _, ok := t.probes[key]; ok {
		return nil
	}

	opts := &link.UprobeOptions{}
	if program.Address != 0 {
		opts.Address = program.Address
	}

	var probe link.Link
	switch program.Type {
	case BpfEventTypeUProbe:
		probe, err = ex.Uprobe(program.Symbol, program.Handler, opts)
	case BpfEventTypeURetProbe:
		probe, err = ex.Uretprobe(program.Symbol, program.Handler, opts)
	default:
		return fmt.Errorf("invalid program type: %s", program.Type)
	}
	if err != nil {
		return fmt.Errorf("error attaching probe: %s", err)
	}

	t.probes[key] = probe

	return nil
}

func (t *Tracer) addTarget(container container.ContainerInfo) error {
	pidNamespaceInode, err := getPIDNamespaceInode(container.Pid)

	if err != nil {
		return errors.Wrapf(err, "failed to get PID namespace inode")
	}

	err = t.targetMap.Update(
		pidNamespaceInode,
		otrzebpf.BpfTargetT{
			Enabled: true,
		},
		ebpf.UpdateAny)

	if err != nil {
		return errors.Wrapf(err, "failed to update target map")
	}

	t.reader.containerMap[pidNamespaceInode] = container

	return nil
}

func (t *Tracer) removeTarget(info container.ContainerInfo) error {
	pidNamespaceInode, err := getPIDNamespaceInode(info.Pid)

	if err != nil {
		return errors.Wrapf(err, "failed to get PID namespace inode")
	}

	err = t.targetMap.Delete(pidNamespaceInode)

	if err != nil {
		return errors.Wrapf(err, "failed to delete target map entry")
	}

	delete(t.reader.containerMap, pidNamespaceInode)

	return nil
}
