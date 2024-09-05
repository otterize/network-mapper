package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/cilium/ebpf/perf"
	"github.com/otterize/network-mapper/src/bintools"
	"github.com/otterize/network-mapper/src/bpfmanager"
	"github.com/otterize/network-mapper/src/ebpf"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	// binPath is the path to the Go binary to inspect.
	binPath = "/Users/davidrobert/ebpf/bin/main"
	// WriteGoTLSFunc is the name of the function that writes to the TLS connection.
	WriteGoTLSFunc = "crypto/tls.(*Conn).Write"
	// ReadGoTLSFunc is the name of the function that reads from the TLS connection.
	ReadGoTLSFunc = "crypto/tls.(*Conn).Read"
)

var FunctionsToProcess = []string{WriteGoTLSFunc, ReadGoTLSFunc}

func B2S(bs []uint8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

func main() {
	inspectionResult, err := bintools.ProcessGoBinary(binPath, FunctionsToProcess)
	if err != nil {
		log.Fatalf("error extracting inspectoin data from %s: %s", binPath, err)
	}

	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	// Load pre-compiled programs and maps into the kernel.
	objs := ebpf.BpfObjects{}
	if err := ebpf.LoadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %s", err)
	}
	defer objs.Close()

	manager := bpfmanager.NewProbeManager(binPath)

	// Register the ebpf programs with the manager.
	manager.RegisterProgram(bpfmanager.BpfProgram{
		Type:    bpfmanager.BpfEventTypeUProbe,
		Symbol:  WriteGoTLSFunc,
		Address: inspectionResult.Functions[WriteGoTLSFunc].EntryAddress,
		Handler: objs.GoTlsWriteEnter,
	})
	manager.RegisterProgram(bpfmanager.BpfProgram{
		Type:    bpfmanager.BpfEventTypeUProbe,
		Symbol:  ReadGoTLSFunc,
		Address: inspectionResult.Functions[ReadGoTLSFunc].EntryAddress,
		Handler: objs.GotlsReadEnter,
	})

	for _, retLoc := range inspectionResult.Functions[ReadGoTLSFunc].ReturnAddresses {
		manager.RegisterProgram(bpfmanager.BpfProgram{
			Type:    bpfmanager.BpfEventTypeUProbe,
			Symbol:  ReadGoTLSFunc,
			Address: retLoc,
			Handler: objs.GotlsReadReturn,
		})
	}

	defer manager.Close()

	err = manager.Init()
	if err != nil {
		log.Fatalf("Failed init manager: %v", err)
	}

	reader, err := perf.NewReader(objs.SslEvents, os.Getpagesize()*64)
	if err != nil {
		log.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	go func() {
		// Wait for a signal and close the perf reader,
		// which will interrupt rd.Read() and make the program exit.
		<-stopper
		log.Println("Received signal, exiting program..")

		if err := reader.Close(); err != nil {
			log.Fatalf("closing perf event reader: %s", err)
		}
	}()

	log.Println("Waiting for events..")

	var event ebpf.BpfSslEventT
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				return
			}
			log.Printf("Error reading from perf event reader: %s", err)
			continue
		}

		if record.LostSamples != 0 {
			log.Printf("Perf event ring buffer full, dropped %d samples", record.LostSamples)
			continue
		}

		// Parse the perf event entry into a bpfEvent structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing perf event: %s", err)
			continue
		}

		msgString := B2S(event.Data[:event.Meta.DataSize])
		log.Printf("  Pid: %d\n", event.Meta.Pid)
		log.Printf("  Msg pos: %d\n", event.Meta.Position)
		log.Printf("  Msg size: %d\n", event.Meta.TotalSize)
		log.Printf("  Msg: %s\n", msgString)
	}
}
