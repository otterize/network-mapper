package ebpf

import (
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/pkg/errors"
)

type probeKey struct {
	inode       uint64
	fnName      string
	retprobe    bool
	programName string
}

type Tracer struct {
	targetMap *ebpf.Map
	probes    map[probeKey]link.Link
	reader    *EventReader
}

func NewTracer(
	reader *EventReader,
) *Tracer {
	t := &Tracer{
		targetMap: openssl.BpfObjects.Maps.Targets,
		probes:    make(map[probeKey]link.Link),
		reader:    reader,
	}

	return t
}

func (t *Tracer) attachToFunction(
	ex *link.Executable,
	binaryInode uint64,
	fnName string,
	retprobe bool,
	program *ebpf.Program,
	programName string,
) error {
	key := getProbeKey(binaryInode, fnName, retprobe, programName)

	if _, ok := t.probes[key]; ok {
		return nil
	}

	var probe link.Link
	var err error

	if retprobe {
		probe, err = ex.Uretprobe(fnName, program, nil)
	} else {
		probe, err = ex.Uprobe(fnName, program, nil)
	}

	if err != nil {
		return errors.Wrapf(err, "failed to attach to %s", fnName)
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
		openssl.SslTargetT{
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
