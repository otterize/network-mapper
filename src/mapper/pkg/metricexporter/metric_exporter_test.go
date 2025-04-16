package metricexporter

import (
	"context"
	"testing"
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type MetricExporterTestSuite struct {
	suite.Suite
	testNamespace  string
	intentsHolder  *intentsstore.IntentsHolder
	edgeMock       *MockEdgeMetric
	metricExporter *MetricExporter
}

var (
	testTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

func (o *MetricExporterTestSuite) SetupTest() {
	o.testNamespace = "default"
	o.intentsHolder = intentsstore.NewIntentsHolder()
}

func (o *MetricExporterTestSuite) BeforeTest(s, testName string) {
	controller := gomock.NewController(o.T())
	o.edgeMock = NewMockEdgeMetric(controller)

	metricExporter, err := NewMetricExporter(context.Background())
	o.Require().NoError(err)
	metricExporter.edgeMetric = o.edgeMock
	o.metricExporter = metricExporter
}

func (o *MetricExporterTestSuite) addIntent(source string, srcNamespace string, destination string, dstNamespace string) {
	o.intentsHolder.AddIntent(
		testTimestamp,
		model.Intent{
			Client: &model.OtterizeServiceIdentity{Name: source, Namespace: srcNamespace},
			Server: &model.OtterizeServiceIdentity{Name: destination, Namespace: dstNamespace},
		},
		make([]int64, 0),
	)
}

func (o *MetricExporterTestSuite) TestExportIntents() {
	o.addIntent("client1", o.testNamespace, "server1", o.testNamespace)
	o.addIntent("client1", o.testNamespace, "server2", "external-namespace")
	o.edgeMock.EXPECT().Record(context.Background(), "client1", "server1").Times(1)
	o.edgeMock.EXPECT().Record(context.Background(), "client1", "server2").Times(1)
	o.metricExporter.NotifyIntents(context.Background(), o.intentsHolder.GetNewIntentsSinceLastGet())
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(MetricExporterTestSuite))
}
