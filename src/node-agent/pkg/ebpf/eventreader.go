package ebpf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/otterize/iamlive/iamlivecore"
	"github.com/otterize/intents-operator/src/shared/errors"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/service"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
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

			err = e.handleEvent(event, data)
			if err != nil {
				logrus.Printf("error handling event: %s", err)
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

func (e *EventReader) handleEvent(event otrzebpf.BpfSslEventT, data []byte) error {
	// Try to parse the event as an HTTP message
	errReq := e.handleHttpRequest(event, data)
	if errReq == nil {
		return nil
	}

	errRes := e.handleHttpResponse(event, data)
	if errRes == nil {
		return nil
	}

	// Try to parse the event as an SMTP message
	// TODO: Implement

	// Try to parse the event as an FTP message
	// TODO: Implement

	return fmt.Errorf("%w\n%w", errReq, errRes)
}

func (e *EventReader) handleHttpRequest(event otrzebpf.BpfSslEventT, data []byte) error {
	reader := bufio.NewReader(bytes.NewReader(data))
	req, err := http.ReadRequest(reader)
	if err != nil {
		return errors.Wrap(err)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return errors.Wrap(err)
	}

	if service.PrintHttpRequests() {
		logrus.Debugf("Got HTTP request: %s\n", string(body))
	}

	// Handle outgoing HTTP requests
	if Direction(event.Meta.Direction) == DirectionEgress {
		pidNamespaceInode, err := getPIDNamespaceInode(int(event.Meta.Pid))
		if err != nil {
			return errors.Errorf("error getting PID namespace inode: %w", err)
		}

		req.RemoteAddr = e.containerMap[pidNamespaceInode].PodIP
		iamlivecore.HandleAWSRequest(req, body, 200)
	}

	return nil
}

func (e *EventReader) handleHttpResponse(event otrzebpf.BpfSslEventT, data []byte) error {
	reader := bufio.NewReader(bytes.NewReader(data))
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return errors.Wrap(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err)
	}

	if service.PrintHttpRequests() {
		logrus.Debugf("Got HTTP response: %s\n", string(body))
	}

	return nil
}

func (e *EventReader) Close() error {
	return e.perfReader.Close()
}
