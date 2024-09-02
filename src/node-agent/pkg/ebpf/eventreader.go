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
	"github.com/sirupsen/logrus"
	"io/ioutil"
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
	perfReader, err := perf.NewReader(perfMap, 4096)
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
			record, err := e.perfReader.Read()

			if err != nil {
				if errors.Is(err, perf.ErrClosed) {
					return
				}

				logrus.Printf("reading from perf event reader: %s", err)
				continue
			}

			if record.LostSamples != 0 {
				logrus.Printf("lost %d samples", record.LostSamples)
				continue
			}

			var event otrzebpf.BpfSslEventT
			byteReader := bytes.NewReader(record.RawSample)

			err = binary.Read(byteReader, binary.LittleEndian, &event.Meta)

			if err != nil {
				logrus.Printf("parsing perf event metadata: %s", err)
				continue
			}

			req, err := http.ReadRequest(bufio.NewReaderSize(byteReader, int(event.Meta.DataSize)))

			pidNamespaceInode, err := getPIDNamespaceInode(int(event.Meta.Pid))

			if err != nil {
				logrus.Errorf("getting PID namespace inode: %s", err)
				continue
			}

			req.RemoteAddr = e.containerMap[pidNamespaceInode].PodIP

			logrus.Printf("HTTP request received %v", req.RemoteAddr)

			if err != nil {
				logrus.Errorf("reading HTTP request: %s", err)
				continue
			}

			reqBody, _ := ioutil.ReadAll(req.Body)

			iamlivecore.HandleAWSRequest(req, reqBody, 200)
			logrus.Println("HTTP request handled", req.RemoteAddr)
		}
	}()
}

func ReadEvents() {
	go func() {
		reader, err := perf.NewReader(otrzebpf.Objs.SslEvents, os.Getpagesize()*64)
		if err != nil {
			logrus.Fatalf("Failed to create reader: %v", err)
		}
		defer reader.Close()

		var event otrzebpf.BpfSslEventT
		for {
			logrus.Debug("Reading go events...")
			record, err := reader.Read()
			if err != nil {
				if errors.Is(err, perf.ErrClosed) {
					return
				}
				logrus.Printf("Error reading from perf event reader: %s", err)
				continue
			}

			if record.LostSamples != 0 {
				logrus.Printf("Perf event ring buffer full, dropped %d samples", record.LostSamples)
				continue
			}

			// Parse the perf event entry into a bpfEvent structure.
			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
				logrus.Printf("parsing perf event: %s", err)
				continue
			}

			msgString := B2S(event.Data[:event.Meta.DataSize])
			logrus.Debug("  Pid: %d\n", event.Meta.Pid)
			logrus.Printf("  Msg pos: %d\n", event.Meta.Position)
			logrus.Printf("  Msg size: %d\n", event.Meta.TotalSize)
			logrus.Printf("  Msg: %s\n", msgString)
		}
	}()
}

func B2S(bs []uint8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

func (e *EventReader) Close() error {
	return e.perfReader.Close()
}
