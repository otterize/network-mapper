// Code generated by bpf2go; DO NOT EDIT.
//go:build arm64

package ebpf

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
)

type BpfGoContextIdT struct {
	Pid  uint64
	Goid uint64
}

type BpfGoSliceT struct {
	Ptr uint64
	Len int32
	Cap int32
}

type BpfSslContextT struct {
	Size   uint64
	Buffer uint64
}

type BpfSslEventT struct {
	Meta struct {
		Pid       uint32
		Position  uint32
		Timestamp uint64
		DataSize  uint32
		TotalSize uint32
	}
	Data [30720]uint8
}

type BpfTargetT struct{ Enabled bool }

// LoadBpf returns the embedded CollectionSpec for Bpf.
func LoadBpf() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_BpfBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load Bpf: %w", err)
	}

	return spec, err
}

// LoadBpfObjects loads Bpf and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//	*BpfObjects
//	*BpfPrograms
//	*BpfMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func LoadBpfObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := LoadBpf()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// BpfSpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type BpfSpecs struct {
	BpfProgramSpecs
	BpfMapSpecs
}

// BpfSpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type BpfProgramSpecs struct {
	GoTlsWriteEnter      *ebpf.ProgramSpec `ebpf:"go_tls_write_enter"`
	GotlsReadEnter       *ebpf.ProgramSpec `ebpf:"gotls_read_enter"`
	GotlsReadReturn      *ebpf.ProgramSpec `ebpf:"gotls_read_return"`
	OtterizeSSL_write    *ebpf.ProgramSpec `ebpf:"otterize_SSL_write"`
	OtterizeSSL_writeRet *ebpf.ProgramSpec `ebpf:"otterize_SSL_write_ret"`
}

// BpfMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type BpfMapSpecs struct {
	GoTlsContext *ebpf.MapSpec `ebpf:"go_tls_context"`
	SslContexts  *ebpf.MapSpec `ebpf:"ssl_contexts"`
	SslEvent     *ebpf.MapSpec `ebpf:"ssl_event"`
	SslEvents    *ebpf.MapSpec `ebpf:"ssl_events"`
	Targets      *ebpf.MapSpec `ebpf:"targets"`
}

// BpfObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to LoadBpfObjects or ebpf.CollectionSpec.LoadAndAssign.
type BpfObjects struct {
	BpfPrograms
	BpfMaps
}

func (o *BpfObjects) Close() error {
	return _BpfClose(
		&o.BpfPrograms,
		&o.BpfMaps,
	)
}

// BpfMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to LoadBpfObjects or ebpf.CollectionSpec.LoadAndAssign.
type BpfMaps struct {
	GoTlsContext *ebpf.Map `ebpf:"go_tls_context"`
	SslContexts  *ebpf.Map `ebpf:"ssl_contexts"`
	SslEvent     *ebpf.Map `ebpf:"ssl_event"`
	SslEvents    *ebpf.Map `ebpf:"ssl_events"`
	Targets      *ebpf.Map `ebpf:"targets"`
}

func (m *BpfMaps) Close() error {
	return _BpfClose(
		m.GoTlsContext,
		m.SslContexts,
		m.SslEvent,
		m.SslEvents,
		m.Targets,
	)
}

// BpfPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to LoadBpfObjects or ebpf.CollectionSpec.LoadAndAssign.
type BpfPrograms struct {
	GoTlsWriteEnter      *ebpf.Program `ebpf:"go_tls_write_enter"`
	GotlsReadEnter       *ebpf.Program `ebpf:"gotls_read_enter"`
	GotlsReadReturn      *ebpf.Program `ebpf:"gotls_read_return"`
	OtterizeSSL_write    *ebpf.Program `ebpf:"otterize_SSL_write"`
	OtterizeSSL_writeRet *ebpf.Program `ebpf:"otterize_SSL_write_ret"`
}

func (p *BpfPrograms) Close() error {
	return _BpfClose(
		p.GoTlsWriteEnter,
		p.GotlsReadEnter,
		p.GotlsReadReturn,
		p.OtterizeSSL_write,
		p.OtterizeSSL_writeRet,
	)
}

func _BpfClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//
//go:embed bpf_arm64_bpfel.o
var _BpfBytes []byte
