package ipresolver

import (
	"github.com/golang/mock/gomock"
	"reflect"
)

type IPResolver interface {
	Refresh() error
	ResolveIP(ipaddr string) (hostname string, err error)
}

// NewMockIPResolver creates a new mock instance.
func NewMockIPResolver(ctrl *gomock.Controller) *MockIPResolver {
	mock := &MockIPResolver{ctrl: ctrl}
	mock.recorder = &MockIPResolverMockRecorder{mock}
	return mock
}

// MockMapperClient is a mock of IPResolver interface.
type MockIPResolver struct {
	ctrl     *gomock.Controller
	recorder *MockIPResolverMockRecorder
}

// MockMapperClientMockRecorder is the mock recorder for MockMapperClient.
type MockIPResolverMockRecorder struct {
	mock *MockIPResolver
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIPResolver) EXPECT() *MockIPResolverMockRecorder {
	return m.recorder
}

// Refresh mocks base method.
func (m *MockIPResolver) Refresh() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Refresh")
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportCaptureResults indicates an expected call of ReportCaptureResults.
func (mr *MockIPResolverMockRecorder) Refresh() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Refresh", reflect.TypeOf((*MockIPResolver)(nil).Refresh))
}

// ResolveIP mocks base method.
func (m *MockIPResolver) ResolveIP(ipaddr string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveIP", ipaddr)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveIP indicates an expected call of ResolveIP.
func (mr *MockIPResolverMockRecorder) ResolveIP(ipaddr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveIP", reflect.TypeOf((*MockIPResolver)(nil).ResolveIP), ipaddr)
}
