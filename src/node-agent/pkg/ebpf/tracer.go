package ebpf

import (
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
	"github.com/pkg/errors"
	"github.com/prometheus/procfs"
)

type Tracer struct {
	pidMap *ebpf.Map
	probes []link.Link
}

func NewTracer() Tracer {
	return Tracer{
		pidMap: openssl.BpfObjects.Maps.PidTargets,
	}
}

func (t *Tracer) AttachToOpenSSL(pid uint32) error {
	fs, err := procfs.NewFS("/host/proc")

	if err != nil {
		return errors.Wrapf(err, "failed to open procfs")
	}

	proc, err := fs.Proc(int(pid))

	if err != nil {
		return errors.Wrapf(err, "failed to open proc %d", pid)
	}

	namespaces, err := proc.Namespaces()

	if err != nil {
		return errors.Wrapf(err, "failed to get namespaces for PID %d", pid)
	}

	pidNamespace := namespaces["pid"]

	libsslPath := fmt.Sprintf("/host/proc/%d/root/lib/aarch64-linux-gnu/libssl.so.3", pid)

	ex, err := link.OpenExecutable(libsslPath)

	if err != nil {
		return errors.Wrapf(err, "failed to open executable %s", libsslPath)
	}

	probe, err := ex.Uprobe("SSL_write", openssl.BpfObjects.OtterizeSSL_write, nil)

	if err != nil {
		return errors.Wrapf(err, "failed to attach to SSL_write")
	}

	t.probes = append(t.probes, probe)

	err = t.pidMap.Update(pidNamespace.Inode, uint8(1), ebpf.UpdateAny)

	if err != nil {
		return errors.Wrapf(err, "failed to update pid map")
	}

	return nil
}
