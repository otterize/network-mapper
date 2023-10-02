package otelexporter

import (
	"context"
	"testing"
	"time"

	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/stretchr/testify/suite"
)

var (
	testTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

type OTelExporterTestSuite struct {
	suite.Suite
	testNamespace string
	intentsHolder *intentsstore.IntentsHolder
	otelExporter  *OtelExporter
}

func (o *OTelExporterTestSuite) SetupTest() {
	o.testNamespace = "default"
	o.intentsHolder = intentsstore.NewIntentsHolder()
}

func (o *OTelExporterTestSuite) BeforeTest(_, testName string) {
	otelExporter, _ := NewOtelExporter(context.Background(), o.intentsHolder, Config{ExportInterval: 1 * time.Second})
	o.otelExporter = otelExporter
}

func (o *OTelExporterTestSuite) addIntent(source string, srcNamespace string, destination string, dstNamespace string) {
	o.intentsHolder.AddIntent(
		testTimestamp,
		model.Intent{
			Client: &model.OtterizeServiceIdentity{Name: source, Namespace: srcNamespace},
			Server: &model.OtterizeServiceIdentity{Name: destination, Namespace: dstNamespace},
		},
	)
}

func (o *OTelExporterTestSuite) TestExportIntents() {
	o.addIntent("client1", o.testNamespace, "server1", o.testNamespace)
	o.addIntent("client1", o.testNamespace, "server2", "external-namespace")

	o.otelExporter.countDiscoveredIntents(context.Background())
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(OTelExporterTestSuite))
}
