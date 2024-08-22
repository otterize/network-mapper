// Code generated by bpf2go; DO NOT EDIT.
//go:build arm64

package openssl

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
)

type opensslSslContextT struct {
	Size   uint64
	Buffer uint64
}

type opensslSslEventT struct {
	Pid       uint32
	_         [4]byte
	Timestamp uint64
	Size      uint32
	Data      [1024]uint8
	_         [4]byte
}

// loadOpenssl returns the embedded CollectionSpec for openssl.
func loadOpenssl() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_OpensslBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load openssl: %w", err)
	}

	return spec, err
}

// loadOpensslObjects loads openssl and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//	*opensslObjects
//	*opensslPrograms
//	*opensslMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func loadOpensslObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadOpenssl()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// opensslSpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type opensslSpecs struct {
	opensslProgramSpecs
	opensslMapSpecs
}

// opensslSpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type opensslProgramSpecs struct {
	OtterizeSSL_write    *ebpf.ProgramSpec `ebpf:"otterize_SSL_write"`
	OtterizeSSL_writeRet *ebpf.ProgramSpec `ebpf:"otterize_SSL_write_ret"`
}

// opensslMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type opensslMapSpecs struct {
	PidTargets  *ebpf.MapSpec `ebpf:"pid_targets"`
	SslContexts *ebpf.MapSpec `ebpf:"ssl_contexts"`
	SslEvent    *ebpf.MapSpec `ebpf:"ssl_event"`
	SslEvents   *ebpf.MapSpec `ebpf:"ssl_events"`
}

// opensslObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to loadOpensslObjects or ebpf.CollectionSpec.LoadAndAssign.
type opensslObjects struct {
	opensslPrograms
	opensslMaps
}

func (o *opensslObjects) Close() error {
	return _OpensslClose(
		&o.opensslPrograms,
		&o.opensslMaps,
	)
}

// opensslMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to loadOpensslObjects or ebpf.CollectionSpec.LoadAndAssign.
type opensslMaps struct {
	PidTargets  *ebpf.Map `ebpf:"pid_targets"`
	SslContexts *ebpf.Map `ebpf:"ssl_contexts"`
	SslEvent    *ebpf.Map `ebpf:"ssl_event"`
	SslEvents   *ebpf.Map `ebpf:"ssl_events"`
}

func (m *opensslMaps) Close() error {
	return _OpensslClose(
		m.PidTargets,
		m.SslContexts,
		m.SslEvent,
		m.SslEvents,
	)
}

// opensslPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to loadOpensslObjects or ebpf.CollectionSpec.LoadAndAssign.
type opensslPrograms struct {
	OtterizeSSL_write    *ebpf.Program `ebpf:"otterize_SSL_write"`
	OtterizeSSL_writeRet *ebpf.Program `ebpf:"otterize_SSL_write_ret"`
}

func (p *opensslPrograms) Close() error {
	return _OpensslClose(
		p.OtterizeSSL_write,
		p.OtterizeSSL_writeRet,
	)
}

func _OpensslClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//
//go:embed openssl_arm64_bpfel.o
var _OpensslBytes []byte
