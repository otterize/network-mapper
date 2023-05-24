package istiowatcher

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient"
	mock_mapperclient "github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient/mockclient"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

// Test istio watcher logic based on testify/suite

type WatcherTestSuite struct {
	suite.Suite
	mockMapperClient *mock_mapperclient.MockMapperClient
	watcher          *IstioWatcher
}

func (s *WatcherTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.mockMapperClient = mock_mapperclient.NewMockMapperClient(controller)
	s.watcher = &IstioWatcher{
		mapperClient: s.mockMapperClient,
		connections:  map[ConnectionWithPath]time.Time{},
		metricsCount: map[string]int{},
	}
}

func generateMetricName(sourceWorkload, sourceNamespace, destinationWorkload, destinationNamespace, requestPath, requestMethod string) string {
	return fmt.Sprintf("istiocustom.istio_requests_total.reporter.source.source_workload.%s.source_canonical_service.client.source_canonical_revision.latest.source_workload_namespace.%s.source_principal.spiffe://cluster.local/ns/test-ns/sa/client-service-account.source_app.client.source_version.unknown.source_cluster.Kubernetes.destination_workload.%s.destination_workload_namespace.%s.destination_principal.spiffe://cluster.local/ns/test-ns/sa/default.destination_app.nginx.destination_version.destination_service.nginx-service.test-ns.svc.cluster.local.destination_canonical_service.nginx.destination_canonical_revision.latest.destination_service_name.nginx-service.destination_service_namespace.test-ns.destination_cluster.Kubernetes.request_protocol.http.response_code.200.grpc_response_status.response_flags.-.connection_security_policy.mutual_tls.request_method.%s.request_path.%s", sourceWorkload, sourceNamespace, destinationWorkload, destinationNamespace, requestMethod, requestPath)
}

func (s *WatcherTestSuite) TestIgnoreOldMetrics() {
	firstMetricsRound := EnvoyMetrics{
		Stats: []Metric{
			{Value: 5, Name: generateMetricName("clientA", "test-ns", "server", "test-ns", "/a-path", "GET")},
			{Value: 1, Name: generateMetricName("clientB", "test-ns", "server", "test-ns", "/b-path", "GET")},
			{Value: 2, Name: generateMetricName("clientC", "test-ns", "server", "test-ns", "/c-path", "POST")},
		},
	}

	secondMetricsRound := EnvoyMetrics{
		Stats: []Metric{
			{Value: 1, Name: generateMetricName("clientA", "test-ns", "server", "test-ns", "/a-path", "GET")},
			{Value: 1, Name: generateMetricName("clientB", "test-ns", "server", "test-ns", "/b-path", "GET")},
			{Value: 3, Name: generateMetricName("clientC", "test-ns", "server", "test-ns", "/c-path", "POST")},
		},
	}
	connectionA := ConnectionWithPath{
		SourceWorkload:       "clientA",
		SourceNamespace:      "test-ns",
		DestinationWorkload:  "server",
		DestinationNamespace: "test-ns",
		RequestPath:          "/a-path",
		RequestMethod:        "GET",
	}
	connectionB := ConnectionWithPath{
		SourceWorkload:       "clientB",
		SourceNamespace:      "test-ns",
		DestinationWorkload:  "server",
		DestinationNamespace: "test-ns",
		RequestPath:          "/b-path",
		RequestMethod:        "GET",
	}
	connectionC := ConnectionWithPath{
		SourceWorkload:       "clientC",
		SourceNamespace:      "test-ns",
		DestinationWorkload:  "server",
		DestinationNamespace: "test-ns",
		RequestPath:          "/c-path",
		RequestMethod:        "POST",
	}

	firstMetricsChannel := make(chan *EnvoyMetrics)
	go func() {
		firstMetricsChannel <- &firstMetricsRound
		close(firstMetricsChannel)
	}()

	err := s.watcher.convertMetricsToConnections(firstMetricsChannel)
	s.NoError(err)
	firstRoundConnections := s.watcher.Flush()
	s.Equal(3, len(firstRoundConnections))
	s.Require().Contains(firstRoundConnections, connectionA)
	s.Require().Contains(firstRoundConnections, connectionB)
	s.Require().Contains(firstRoundConnections, connectionC)

	secondMetricsChannel := make(chan *EnvoyMetrics)
	go func() {
		secondMetricsChannel <- &secondMetricsRound
		close(secondMetricsChannel)
	}()

	err = s.watcher.convertMetricsToConnections(secondMetricsChannel)
	s.NoError(err)
	secondRoundConnections := s.watcher.Flush()
	s.Require().Equal(2, len(secondRoundConnections))

	s.Require().Contains(secondRoundConnections, connectionA)
	s.Require().Contains(secondRoundConnections, connectionC)
	s.Require().NotContains(secondRoundConnections, connectionB)
}

func (s *WatcherTestSuite) TestReportResults() {
	metrics := EnvoyMetrics{
		Stats: []Metric{
			{Value: 5, Name: generateMetricName("clientA", "test-ns", "server", "test-ns", "/a-path", "GET")},
			{Value: 1, Name: generateMetricName("clientB", "test-ns", "server", "test-ns", "/b-path", "GET")},
			{Value: 2, Name: generateMetricName("clientA", "test-ns", "server", "test-ns", "/a-path", "POST")},
		},
	}

	connectionA := mapperclient.IstioConnection{
		SrcWorkload:          "clientA",
		SrcWorkloadNamespace: "test-ns",
		DstWorkload:          "server",
		DstWorkloadNamespace: "test-ns",
		Path:                 "/a-path",
		Methods:              []mapperclient.HttpMethod{mapperclient.HttpMethodGet, mapperclient.HttpMethodPost},
		LastSeen:             time.Time{},
	}

	connectionB := mapperclient.IstioConnection{
		SrcWorkload:          "clientB",
		SrcWorkloadNamespace: "test-ns",
		DstWorkload:          "server",
		DstWorkloadNamespace: "test-ns",
		Path:                 "/b-path",
		Methods:              []mapperclient.HttpMethod{mapperclient.HttpMethodGet},
		LastSeen:             time.Time{},
	}

	firstMetricsChannel := make(chan *EnvoyMetrics)
	go func() {
		firstMetricsChannel <- &metrics
		close(firstMetricsChannel)
	}()

	err := s.watcher.convertMetricsToConnections(firstMetricsChannel)
	s.NoError(err)

	istioConnections := mapperclient.IstioConnectionResults{
		Results: []mapperclient.IstioConnection{
			connectionA,
			connectionB,
		},
	}

	s.mockMapperClient.EXPECT().ReportIstioConnections(gomock.Any(), GetMatcher(istioConnections)).Return(nil)
	err = s.watcher.reportResults(context.Background())
	s.NoError(err)
}

func TestWatcher(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
