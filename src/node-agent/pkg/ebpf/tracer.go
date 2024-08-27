package ebpf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
	"github.com/pkg/errors"
	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	"os"
	"syscall"
)

type probeKey struct {
	inode       uint64
	fnName      string
	retprobe    bool
	programName string
}

type Tracer struct {
	pidMap *ebpf.Map
	probes map[probeKey]link.Link
}

func NewTracer() Tracer {
	t := Tracer{
		pidMap: openssl.BpfObjects.Maps.Targets,
		probes: make(map[probeKey]link.Link),
	}

	rd, err := perf.NewReader(openssl.BpfObjects.SslEvents, os.Getpagesize())

	if err != nil {
		logrus.Fatalf("creating perf event reader: %s", err)
	}

	go func() {
		for {
			record, err := rd.Read()

			if err != nil {
				if errors.Is(err, perf.ErrClosed) {
					return
				}

				logrus.Printf("reading from perf event reader: %s", err)
				continue
			}

			var event openssl.SslEventT

			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
				logrus.Printf("parsing perf event: %s", err)
				continue
			}

			logrus.Printf("Received event: %v", event.Data)
		}
	}()

	return t
}

func (t *Tracer) AttachToOpenSSL(pid int) error {
	libsslPath := fmt.Sprintf("/host/proc/%d/root/lib/aarch64-linux-gnu/libssl.so.3", pid)

	ex, err := link.OpenExecutable(libsslPath)

	if err != nil {
		return errors.Wrapf(err, "failed to open executable %s", libsslPath)
	}

	inode, err := getFileInode(libsslPath)

	if err != nil {
		return errors.Wrapf(err, "failed to get inode for %s", libsslPath)
	}

	pidNamespaceInode, err := getPIDNamespaceInode(pid)

	err = t.pidMap.Update(pidNamespaceInode, uint8(1), ebpf.UpdateAny)

	if err != nil {
		return errors.Wrapf(err, "failed to update pid map")
	}

	err = t.attachToFunction(
		ex,
		inode,
		"SSL_write",
		false,
		openssl.BpfObjects.OtterizeSSL_write,
		openssl.BpfSpecs.OtterizeSSL_write.Name,
	)

	if err != nil {
		return errors.Wrapf(err, "failed to attach to SSL_write")
	}

	err = t.attachToFunction(
		ex,
		inode,
		"SSL_write",
		true,
		openssl.BpfObjects.OtterizeSSL_writeRet,
		openssl.BpfSpecs.OtterizeSSL_writeRet.Name,
	)

	if err != nil {
		return errors.Wrapf(err, "failed to attach to SSL_write (ret)")
	}

	return nil
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

func getProbeKey(
	binaryInode uint64,
	fnName string,
	retprobe bool,
	programName string,
) probeKey {
	return probeKey{
		inode:       binaryInode,
		fnName:      fnName,
		retprobe:    retprobe,
		programName: programName,
	}
}

func getFileInode(path string) (uint64, error) {
	stat, err := os.Stat(path)

	if err != nil {
		return 0, errors.Wrapf(err, "failed to stat %s", path)
	}

	ino, ok := stat.Sys().(*syscall.Stat_t)

	if !ok {
		return 0, errors.New("failed to get inode information for " + path)
	}

	return ino.Ino, nil
}

func getPIDNamespaceInode(pid int) (uint32, error) {
	fs, err := procfs.NewFS("/host/proc")

	if err != nil {
		return 0, errors.Wrapf(err, "failed to open procfs")
	}

	proc, err := fs.Proc(pid)

	if err != nil {
		return 0, errors.Wrapf(err, "failed to open proc %d", pid)
	}

	namespaces, err := proc.Namespaces()

	if err != nil {
		return 0, errors.Wrapf(err, "failed to get namespaces for PID %d", pid)
	}

	pidNamespace, ok := namespaces["pid"]

	if !ok {
		return 0, errors.New("failed to find PID namespace for PID " + string(pid))
	}

	return pidNamespace.Inode, nil
}
