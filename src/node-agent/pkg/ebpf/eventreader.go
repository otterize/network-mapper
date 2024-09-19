package ebpf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/otterize/iamlive/iamlivecore"
	"github.com/otterize/intents-operator/src/shared/errors"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

type EventReader struct {
	perfReader   *perf.Reader
	containerMap map[uint32]container.ContainerInfo
}

func init() {
	iamlivecore.ParseConfig()
	iamlivecore.LoadMaps()
	iamlivecore.ReadServiceFiles()
}

func NewEventReader(perfMap *ebpf.Map) (*EventReader, error) {
	perfReader, err := perf.NewReader(perfMap, os.Getpagesize()*64)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &EventReader{
		perfReader:   perfReader,
		containerMap: make(map[uint32]container.ContainerInfo),
	}, nil
}

func (e *EventReader) Start() {
	go func() {
		for {
			// Read the next TLS event.
			record, err := e.perfReader.Read()
			if err != nil {
				if errors.Is(err, perf.ErrClosed) {
					return
				}
				logrus.Printf("error reading from perf event reader: %s", err)
				continue
			}

			if record.LostSamples != 0 {
				logrus.Printf("lost %d samples", record.LostSamples)
				continue
			}

			// Parse the perf event entry into a bpfEvent structure.
			event, data, err := e.parseEvent(record.RawSample)
			if err != nil {
				logrus.Printf("error parsing perf event: %s", err)
				continue
			}

			// Get the container information for the event
			pidNamespaceInode, err := getPIDNamespaceInode(int(event.Meta.Pid))
			if err != nil {
				logrus.Printf("error getting PID namespace inode: %w", err)
				continue
			}

			// Process the event
			cInfo := e.containerMap[pidNamespaceInode]
			eventContext := types.EventContext{
				Event:     event,
				Data:      data,
				Container: cInfo,
				Metadata:  &types.EventMetadata{},
			}
			err = eventparser.ProcessEvent(eventContext)
			if err != nil {
				logrus.Printf("error processing event: %s", err)
				continue
			}
		}
	}()
}

func (e *EventReader) parseEvent(raw []byte) (otrzebpf.BpfSslEventT, []byte, error) {
	var event otrzebpf.BpfSslEventT

	// Parse the perf event entry into a bpfEvent structure.
	byteReader := bytes.NewReader(raw)
	if err := binary.Read(byteReader, binary.LittleEndian, &event.Meta); err != nil {
		return event, nil, errors.Wrap(err)
	}

	// Read the exact amount of data for the event.
	dataBuffer := bufio.NewReaderSize(byteReader, int(event.Meta.DataSize))
	data, err := io.ReadAll(dataBuffer)
	if err != nil {
		return event, nil, errors.Wrap(err)
	}

	return event, data, nil
}

func (e *EventReader) Close() error {
	return e.perfReader.Close()
}
