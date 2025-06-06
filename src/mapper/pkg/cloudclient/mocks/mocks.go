// Code generated by MockGen. DO NOT EDIT.
// Source: ./cloud_client.go

// Package cloudclientmocks is a generated GoMock package.
package cloudclientmocks

import (
	context "context"
	reflect "reflect"

	cloudclient "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	gomock "go.uber.org/mock/gomock"
)

// MockCloudClient is a mock of CloudClient interface.
type MockCloudClient struct {
	ctrl     *gomock.Controller
	recorder *MockCloudClientMockRecorder
}

// MockCloudClientMockRecorder is the mock recorder for MockCloudClient.
type MockCloudClientMockRecorder struct {
	mock *MockCloudClient
}

// NewMockCloudClient creates a new mock instance.
func NewMockCloudClient(ctrl *gomock.Controller) *MockCloudClient {
	mock := &MockCloudClient{ctrl: ctrl}
	mock.recorder = &MockCloudClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCloudClient) EXPECT() *MockCloudClientMockRecorder {
	return m.recorder
}

// ReportCiliumClusterWideNetworkPolicies mocks base method.
func (m *MockCloudClient) ReportCiliumClusterWideNetworkPolicies(ctx context.Context, policies []cloudclient.NetworkPolicyInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportCiliumClusterWideNetworkPolicies", ctx, policies)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportCiliumClusterWideNetworkPolicies indicates an expected call of ReportCiliumClusterWideNetworkPolicies.
func (mr *MockCloudClientMockRecorder) ReportCiliumClusterWideNetworkPolicies(ctx, policies interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportCiliumClusterWideNetworkPolicies", reflect.TypeOf((*MockCloudClient)(nil).ReportCiliumClusterWideNetworkPolicies), ctx, policies)
}

// ReportComponentStatus mocks base method.
func (m *MockCloudClient) ReportComponentStatus(ctx context.Context, component cloudclient.ComponentType) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportComponentStatus", ctx, component)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportComponentStatus indicates an expected call of ReportComponentStatus.
func (mr *MockCloudClientMockRecorder) ReportComponentStatus(ctx, component interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportComponentStatus", reflect.TypeOf((*MockCloudClient)(nil).ReportComponentStatus), ctx, component)
}

// ReportDiscoveredIntents mocks base method.
func (m *MockCloudClient) ReportDiscoveredIntents(ctx context.Context, intents []*cloudclient.DiscoveredIntentInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportDiscoveredIntents", ctx, intents)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportDiscoveredIntents indicates an expected call of ReportDiscoveredIntents.
func (mr *MockCloudClientMockRecorder) ReportDiscoveredIntents(ctx, intents interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportDiscoveredIntents", reflect.TypeOf((*MockCloudClient)(nil).ReportDiscoveredIntents), ctx, intents)
}

// ReportExternalTrafficDiscoveredIntents mocks base method.
func (m *MockCloudClient) ReportExternalTrafficDiscoveredIntents(ctx context.Context, intents []cloudclient.ExternalTrafficDiscoveredIntentInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportExternalTrafficDiscoveredIntents", ctx, intents)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportExternalTrafficDiscoveredIntents indicates an expected call of ReportExternalTrafficDiscoveredIntents.
func (mr *MockCloudClientMockRecorder) ReportExternalTrafficDiscoveredIntents(ctx, intents interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportExternalTrafficDiscoveredIntents", reflect.TypeOf((*MockCloudClient)(nil).ReportExternalTrafficDiscoveredIntents), ctx, intents)
}

// ReportIncomingTrafficDiscoveredIntents mocks base method.
func (m *MockCloudClient) ReportIncomingTrafficDiscoveredIntents(ctx context.Context, intents []cloudclient.IncomingTrafficDiscoveredIntentInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportIncomingTrafficDiscoveredIntents", ctx, intents)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportIncomingTrafficDiscoveredIntents indicates an expected call of ReportIncomingTrafficDiscoveredIntents.
func (mr *MockCloudClientMockRecorder) ReportIncomingTrafficDiscoveredIntents(ctx, intents interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportIncomingTrafficDiscoveredIntents", reflect.TypeOf((*MockCloudClient)(nil).ReportIncomingTrafficDiscoveredIntents), ctx, intents)
}

// ReportK8sIngresses mocks base method.
func (m *MockCloudClient) ReportK8sIngresses(ctx context.Context, namespace string, ingresses []cloudclient.K8sIngressInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportK8sIngresses", ctx, namespace, ingresses)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportK8sIngresses indicates an expected call of ReportK8sIngresses.
func (mr *MockCloudClientMockRecorder) ReportK8sIngresses(ctx, namespace, ingresses interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportK8sIngresses", reflect.TypeOf((*MockCloudClient)(nil).ReportK8sIngresses), ctx, namespace, ingresses)
}

// ReportK8sResourceEligibleForMetricsCollection mocks base method.
func (m *MockCloudClient) ReportK8sResourceEligibleForMetricsCollection(ctx context.Context, namespace string, reason cloudclient.EligibleForMetricsCollectionReason, resources []cloudclient.K8sResourceEligibleForMetricsCollectionInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportK8sResourceEligibleForMetricsCollection", ctx, namespace, reason, resources)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportK8sResourceEligibleForMetricsCollection indicates an expected call of ReportK8sResourceEligibleForMetricsCollection.
func (mr *MockCloudClientMockRecorder) ReportK8sResourceEligibleForMetricsCollection(ctx, namespace, reason, resources interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportK8sResourceEligibleForMetricsCollection", reflect.TypeOf((*MockCloudClient)(nil).ReportK8sResourceEligibleForMetricsCollection), ctx, namespace, reason, resources)
}

// ReportK8sServices mocks base method.
func (m *MockCloudClient) ReportK8sServices(ctx context.Context, namespace string, services []cloudclient.K8sServiceInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportK8sServices", ctx, namespace, services)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportK8sServices indicates an expected call of ReportK8sServices.
func (mr *MockCloudClientMockRecorder) ReportK8sServices(ctx, namespace, services interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportK8sServices", reflect.TypeOf((*MockCloudClient)(nil).ReportK8sServices), ctx, namespace, services)
}

// ReportK8sWebhookServices mocks base method.
func (m *MockCloudClient) ReportK8sWebhookServices(ctx context.Context, services []cloudclient.K8sWebhookServiceInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportK8sWebhookServices", ctx, services)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportK8sWebhookServices indicates an expected call of ReportK8sWebhookServices.
func (mr *MockCloudClientMockRecorder) ReportK8sWebhookServices(ctx, services interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportK8sWebhookServices", reflect.TypeOf((*MockCloudClient)(nil).ReportK8sWebhookServices), ctx, services)
}

// ReportNamespaceLabels mocks base method.
func (m *MockCloudClient) ReportNamespaceLabels(ctx context.Context, namespace string, labels []cloudclient.LabelInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportNamespaceLabels", ctx, namespace, labels)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportNamespaceLabels indicates an expected call of ReportNamespaceLabels.
func (mr *MockCloudClientMockRecorder) ReportNamespaceLabels(ctx, namespace, labels interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportNamespaceLabels", reflect.TypeOf((*MockCloudClient)(nil).ReportNamespaceLabels), ctx, namespace, labels)
}

// ReportNetworkPolicies mocks base method.
func (m *MockCloudClient) ReportNetworkPolicies(ctx context.Context, namespace string, policies []cloudclient.NetworkPolicyInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportNetworkPolicies", ctx, namespace, policies)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportNetworkPolicies indicates an expected call of ReportNetworkPolicies.
func (mr *MockCloudClientMockRecorder) ReportNetworkPolicies(ctx, namespace, policies interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportNetworkPolicies", reflect.TypeOf((*MockCloudClient)(nil).ReportNetworkPolicies), ctx, namespace, policies)
}

// ReportTrafficLevels mocks base method.
func (m *MockCloudClient) ReportTrafficLevels(ctx context.Context, trafficLevels []cloudclient.TrafficLevelInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportTrafficLevels", ctx, trafficLevels)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportTrafficLevels indicates an expected call of ReportTrafficLevels.
func (mr *MockCloudClientMockRecorder) ReportTrafficLevels(ctx, trafficLevels interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportTrafficLevels", reflect.TypeOf((*MockCloudClient)(nil).ReportTrafficLevels), ctx, trafficLevels)
}

// ReportWorkloadsMetadata mocks base method.
func (m *MockCloudClient) ReportWorkloadsMetadata(ctx context.Context, workloadsLabels []cloudclient.ReportServiceMetadataInput) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportWorkloadsMetadata", ctx, workloadsLabels)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReportWorkloadsMetadata indicates an expected call of ReportWorkloadsMetadata.
func (mr *MockCloudClientMockRecorder) ReportWorkloadsMetadata(ctx, workloadsLabels interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportWorkloadsMetadata", reflect.TypeOf((*MockCloudClient)(nil).ReportWorkloadsMetadata), ctx, workloadsLabels)
}
