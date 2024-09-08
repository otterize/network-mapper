package ebpf

import (
	"fmt"
	"github.com/cilium/ebpf/link"
	"github.com/otterize/intents-operator/src/shared/errors"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/sirupsen/logrus"
)

var progs = []BpfProgram{
	{BpfEventTypeUProbe, "SSL_write", otrzebpf.Objs.OtterizeSSL_write, otrzebpf.Specs.OtterizeSSL_write, 0},
	{BpfEventTypeURetProbe, "SSL_write", otrzebpf.Objs.OtterizeSSL_writeRet, otrzebpf.Specs.OtterizeSSL_writeRet, 0},
	{BpfEventTypeUProbe, "SSL_write_ex", otrzebpf.Objs.OtterizeSSL_write, otrzebpf.Specs.OtterizeSSL_write, 0},
	{BpfEventTypeURetProbe, "SSL_write_ex", otrzebpf.Objs.OtterizeSSL_writeRet, otrzebpf.Specs.OtterizeSSL_writeRet, 0},
}

func (t *Tracer) AttachToOpenSSL(cInfo container.ContainerInfo) error {
	libsslPath := getLibSslSo3Path(cInfo.Pid)

	ex, err := link.OpenExecutable(libsslPath)
	if err != nil {
		return errors.Wrap(err)
	}

	inode, err := getFileInode(libsslPath)
	if err != nil {
		return errors.Wrap(err)
	}

	err = t.addTarget(cInfo)
	if err != nil {
		return errors.Wrap(err)
	}

	for _, prog := range progs {
		logrus.WithField("pid", cInfo.Pid).WithField("symbol", prog.Symbol).WithField("file", libsslPath).Debug("Attaching openssl probe")
		err = t.attachBpfProgram(ex, inode, prog)

		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func getLibSslSo3Path(pid int) string {
	return fmt.Sprintf("/host/proc/%d/root/usr/local/bin/node", pid)
}
