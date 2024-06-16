// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

// Package mock_mapperclient is a generated GoMock package.
package mock_mapperclient

import (
	context "context"
	reflect "reflect"

	mapperclient "github.com/otterize/network-mapper/src/kafka-watcher/pkg/mapperclient"
	gomock "go.uber.org/mock/gomock"
)

// MockMapperClient is a mock of MapperClient interface.
type MockMapperClient struct {
	ctrl     *gomock.Controller
	recorder *MockMapperClientMockRecorder
}

// MockMapperClientMockRecorder is the mock recorder for MockMapperClient.
type MockMapperClientMockRecorder struct {
	mock *MockMapperClient
}

// NewMockMapperClient creates a new mock instance.
func NewMockMapperClient(ctrl *gomock.Controller) *MockMapperClient {
	mock := &MockMapperClient{ctrl: ctrl}
	mock.recorder = &MockMapperClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMapperClient) EXPECT() *MockMapperClientMockRecorder {
	return m.recorder
}

// Health mocks base method.
func (m *MockMapperClient) Health(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Health", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Health indicates an expected call of Health.
func (mr *MockMapperClientMockRecorder) Health(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Health", reflect.TypeOf((*MockMapperClient)(nil).Health), ctx)
}

// ReportKafkaMapperResults mocks base method.
func (m *MockMapperClient) ReportKafkaMapperResults(ctx context.Context, results mapperclient.KafkaMapperResults) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportKafkaMapperResults", ctx, results)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportKafkaMapperResults indicates an expected call of ReportKafkaMapperResults.
func (mr *MockMapperClientMockRecorder) ReportKafkaMapperResults(ctx, results interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportKafkaMapperResults", reflect.TypeOf((*MockMapperClient)(nil).ReportKafkaMapperResults), ctx, results)
}
