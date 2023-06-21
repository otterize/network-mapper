package logwatcher

import (
	"context"
	"errors"
	"github.com/oriser/regroup"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/mapperclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

// AclAuthorizerRegex matches & decodes AclAuthorizer log records.
// Sample log record for reference:
// [2023-03-12 13:51:55,904] INFO Principal = User:2.5.4.45=#13206331373734376636373865323137613636346130653335393130326638303662,CN=myclient.otterize-tutorial-kafka-mtls,O=SPIRE,C=US is Denied Operation = Describe from host = 10.244.0.27 on resource = Topic:LITERAL:mytopic for request = Metadata with resourceRefCount = 1 (kafka.authorizer.logger)
var AclAuthorizerRegex = regroup.MustCompile(
	`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d+\] [A-Z]+ Principal = \S+ is (?P<access>\S+) Operation = (?P<operation>\S+) from host = (?P<host>\S+) on resource = Topic:LITERAL:(?P<topic>.+) for request = \S+ with resourceRefCount = \d+ \(kafka\.authorizer\.logger\)$`,
)

type AuthorizerRecord struct {
	Server    types.NamespacedName
	Access    string `regroup:"access"`
	Operation string `regroup:"operation"`
	Host      string `regroup:"host"`
	Topic     string `regroup:"topic"`
}

type SeenRecordsStore map[AuthorizerRecord]time.Time

type Watcher interface {
	RunForever(ctx context.Context)
}

type baseWatcher struct {
	mu           sync.Mutex
	seen         SeenRecordsStore
	mapperClient mapperclient.MapperClient
}

func (b *baseWatcher) flush() SeenRecordsStore {
	b.mu.Lock()
	defer b.mu.Unlock()
	r := b.seen
	b.seen = SeenRecordsStore{}
	return r
}

func (b *baseWatcher) reportResults(ctx context.Context) error {
	records := b.flush()

	cRecords := len(records)

	if cRecords == 0 {
		logrus.Infof("Zero records, not reporting")
		return nil
	}

	logrus.Infof("Reporting %d records", cRecords)

	results := lo.MapToSlice(records, func(r AuthorizerRecord, t time.Time) mapperclient.KafkaMapperResult {
		return mapperclient.KafkaMapperResult{
			SrcIp:           r.Host,
			ServerPodName:   r.Server.Name,
			ServerNamespace: r.Server.Namespace,
			Topic:           r.Topic,
			Operation:       r.Operation,
			LastSeen:        t,
		}
	})

	return b.mapperClient.ReportKafkaMapperResults(ctx, mapperclient.KafkaMapperResults{Results: results})
}

func (b *baseWatcher) processLogRecord(kafkaServer types.NamespacedName, record string) {
	authorizerRecord := AuthorizerRecord{
		Server: kafkaServer,
	}
	if err := AclAuthorizerRegex.MatchToTarget(record, &authorizerRecord); errors.Is(err, &regroup.NoMatchFoundError{}) {
		return
	} else if err != nil {
		logrus.Errorf("Error matching authorizer regex: %s", err)
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.seen[authorizerRecord] = time.Now()
}
