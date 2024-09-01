package ebpf

import (
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/sirupsen/logrus"
)

type program struct {
	fnName      string
	program     *ebpf.Program
	programSpec *ebpf.ProgramSpec
	retprobe    bool
}

var sslPrograms = []program{
	{"SSL_write", openssl.BpfObjects.OtterizeSSL_write, openssl.BpfSpecs.OtterizeSSL_write, false},
	{"SSL_write", openssl.BpfObjects.OtterizeSSL_writeRet, openssl.BpfSpecs.OtterizeSSL_writeRet, true},
	{"SSL_write_ex", openssl.BpfObjects.OtterizeSSL_write, openssl.BpfSpecs.OtterizeSSL_write, false},
	{"SSL_write_ex", openssl.BpfObjects.OtterizeSSL_writeRet, openssl.BpfSpecs.OtterizeSSL_writeRet, true},
}

func (t *Tracer) AttachToOpenSSL(container container.ContainerInfo) error {
	libsslPath := getLibSslSo3Path(container.Pid)

	ex, err := link.OpenExecutable(libsslPath)

	if err != nil {
		logrus.Errorf("failed to open executable %s: %d", libsslPath, container.Pid)
		return nil
	}

	inode, err := getFileInode(libsslPath)

	if err != nil {
		return errors.Wrap(err)
	}

	err = t.addTarget(container)

	if err != nil {
		return errors.Wrap(err)
	}

	for _, prog := range sslPrograms {
		err = t.attachToFunction(
			ex,
			inode,
			prog.fnName,
			prog.retprobe,
			prog.program,
			prog.programSpec.Name,
		)

		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func getLibSslSo3Path(pid int) string {
	//var arch string
	//if runtime.GOARCH == "amd64" {
	//	arch = "x86_64"
	//} else if runtime.GOARCH == "arm64" {
	//	arch = "aarch64"
	//} else {
	//	arch = "unknown"
	//}

	return fmt.Sprintf("/host/proc/%d/root/usr/local/bin/node", pid)
}
