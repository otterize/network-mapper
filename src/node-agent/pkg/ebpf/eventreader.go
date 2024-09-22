package ebpf

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/otterize/iamlive/iamlivecore"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	otrzebpf "github.com/otterize/network-mapper/src/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"github.com/otterize/network-mapper/src/node-agent/pkg/eventparser"
	"github.com/otterize/network-mapper/src/shared/cloudclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"io"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type EventReader struct {
	client            client.Client
	perfReader        *perf.Reader
	cloudClient       cloudclient.CloudClient
	containerMap      map[uint32]container.ContainerInfo
	serviceIdResolver *serviceidresolver.Resolver
}

func init() {
	iamlivecore.ParseConfig()
	iamlivecore.LoadMaps()
	iamlivecore.ReadServiceFiles()
}

func NewEventReader(
	client client.Client,
	cloudClient cloudclient.CloudClient,
	serviceIdResolver *serviceidresolver.Resolver,
	perfMap *ebpf.Map,
) (*EventReader, error) {
	perfReader, err := perf.NewReader(perfMap, os.Getpagesize()*64)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &EventReader{
		client:            client,
		perfReader:        perfReader,
		cloudClient:       cloudClient,
		containerMap:      make(map[uint32]container.ContainerInfo),
		serviceIdResolver: serviceIdResolver,
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

			// Update the workload with the metadata
			err = e.updateWorkload(eventContext)
			if err != nil {
				logrus.Printf("error updating workload: %s", err)
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

func (e *EventReader) updateWorkload(eventCtx types.EventContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pod := corev1.Pod{}
	if err := e.client.Get(ctx, eventCtx.Container.PodName, &pod); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err)
	}

	serviceId, err := e.serviceIdResolver.ResolvePodToServiceIdentity(ctx, &pod)
	if err != nil {
		return errors.Wrap(err)
	}

	logrus.Printf("SERVICE ID: %v", serviceId)

	// Update the workload with the metadata
	serviceMeta := cloudclient.ReportServiceMetadataInput{
		Identity: cloudclient.ServiceIdentityInput{
			Name:      serviceId.Name,
			Namespace: serviceId.Namespace,
			Kind:      serviceId.Kind,
		},
		Metadata: cloudclient.ServiceMetadataInput{
			Tags: lo.MapToSlice(eventCtx.Metadata.Tags, func(key types.EventTag, _ bool) string {
				return string(key)
			}),
		},
	}

	err = e.cloudClient.ReportServiceMeta(ctx, serviceMeta)
	if err != nil {
		return errors.Wrap(err)

	}

	return nil
}

func (e *EventReader) Close() error {
	return e.perfReader.Close()
}
