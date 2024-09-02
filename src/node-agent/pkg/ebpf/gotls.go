package ebpf

import (
	"fmt"
	"github.com/cilium/ebpf/link"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/bintools"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	// WriteGoTLSFunc is the name of the function that writes to the TLS connection.
	WriteGoTLSFunc = "crypto/tls.(*Conn).Write"
	// ReadGoTLSFunc is the name of the function that reads from the TLS connection.
	ReadGoTLSFunc = "crypto/tls.(*Conn).Read"
)

var FunctionsToProcess = []string{WriteGoTLSFunc, ReadGoTLSFunc}

func (t *Tracer) AttachToGoTls(cInfo container.ContainerInfo) error {
	// Get the path to the Go binary to inspect.
	execPath := container.GetContainerExecPath(cInfo.Pid)
	absPath, err := os.Readlink(execPath)
	if err != nil {
		return errors.Wrap(err)
	}
	binPath := fmt.Sprintf("/host/proc/%d/root%s", cInfo.Pid, absPath)

	// Dynamically calculate the programs based on the binary
	programs, err := getBpfPrograms(otrzebpf.Objs, binPath)
	if err != nil {
		return errors.Wrap(err)
	}

	ex, err := link.OpenExecutable(binPath)
	if err != nil {
		logrus.Errorf("failed to open executable %s: %d", binPath, cInfo.Pid)
		return nil
	}

	inode, err := getFileInode(binPath)
	if err != nil {
		return errors.Wrap(err)
	}

	err = t.addTarget(cInfo)
	if err != nil {
		return errors.Wrap(err)
	}

	for _, prog := range programs {
		logrus.WithField("adr", prog.Address).WithField("symbol", prog.Symbol).WithField("file", binPath).Debug("Attaching gotls probe")
		err = t.attachBpfProgram(ex, inode, prog)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func getBpfPrograms(objs otrzebpf.BpfObjects, binPath string) ([]BpfProgram, error) {
	inspectionResult, err := bintools.ProcessGoBinary(binPath, FunctionsToProcess)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	programs := make([]BpfProgram, 0)
	programs = append(programs, BpfProgram{
		Type:        BpfEventTypeUProbe,
		Symbol:      WriteGoTLSFunc,
		Address:     inspectionResult.Functions[WriteGoTLSFunc].EntryAddress,
		Handler:     objs.GoTlsWriteEnter,
		HandlerSpec: otrzebpf.Specs.GoTlsWriteEnter,
	})
	programs = append(programs, BpfProgram{
		Type:        BpfEventTypeUProbe,
		Symbol:      ReadGoTLSFunc,
		Address:     inspectionResult.Functions[ReadGoTLSFunc].EntryAddress,
		Handler:     objs.GotlsReadEnter,
		HandlerSpec: otrzebpf.Specs.GotlsReadEnter,
	})

	for _, retLoc := range inspectionResult.Functions[ReadGoTLSFunc].ReturnAddresses {
		programs = append(programs, BpfProgram{
			Type:        BpfEventTypeUProbe,
			Symbol:      ReadGoTLSFunc,
			Address:     retLoc,
			Handler:     objs.GotlsReadReturn,
			HandlerSpec: otrzebpf.Specs.GotlsReadReturn,
		})
	}

	return programs, nil
}
