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
		var event otrzebpf.BpfSslEventT
		for {
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
			byteReader := bytes.NewReader(record.RawSample)
			if err = binary.Read(byteReader, binary.LittleEndian, &event); err != nil {
				logrus.Printf("parsing perf event: %s", err)
				continue
			}

			msg := Data2Bytes(event.Data[:event.Meta.DataSize])
			reader := bufio.NewReader(bytes.NewReader(msg))

			var body []byte

			switch Direction(event.Meta.Direction) {
			case DirectionEgress:
				req, err := http.ReadRequest(reader)
				if err != nil {
					logrus.Printf("parsing HTTP request: %s", err)
					continue
				}

				body, err = io.ReadAll(req.Body)
				if err != nil {
					logrus.Printf("error reading HTTP request body: %s", err)
					continue
				}
			case DirectionIngress:
				resp, err := http.ReadResponse(reader, nil)
				if err != nil {
					logrus.Printf("parsing HTTP response: %s", err)
					continue
				}

				body, err = io.ReadAll(resp.Body)
				if err != nil {
					logrus.Printf("error reading HTTP response body: %s", err)
					continue
				}
			default:
				continue
			}
			logrus.Debug("Msg: %s\n", string(body))

			//pidNamespaceInode, err := getPIDNamespaceInode(int(event.Meta.Pid))
			//if err != nil {
			//	logrus.Errorf("getting PID namespace inode: %s", err)
			//	continue
			//}
			//
			//req.RemoteAddr = e.containerMap[pidNamespaceInode].PodIP
			//
			//logrus.Printf("HTTP request received %v", req.RemoteAddr)
			//
			//if err != nil {
			//	logrus.Errorf("reading HTTP request: %s", err)
			//	continue
			//}
			//
			//reqBody, _ := ioutil.ReadAll(req.Body)
			//
			//iamlivecore.HandleAWSRequest(req, reqBody, 200)
			//logrus.Println("HTTP request handled", req.RemoteAddr)
		}
	}()
}

func (e *EventReader) Close() error {
	return e.perfReader.Close()
}
