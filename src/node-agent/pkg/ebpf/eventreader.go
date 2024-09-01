package ebpf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/otterize/iamlive/iamlivecore"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
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

func NewEventReader(
	perfMap *ebpf.Map,
) (*EventReader, error) {
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

			var event openssl.SslEventT
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

func (e *EventReader) Close() error {
	return e.perfReader.Close()
}
