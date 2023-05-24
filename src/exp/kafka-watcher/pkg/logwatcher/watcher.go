package logwatcher

import (
	"context"
	"errors"
	"github.com/nxadm/tail"
	"github.com/oriser/regroup"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/mapperclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

type Watcher struct {
	mu            sync.Mutex
	seen          SeenRecordsStore
	mapperClient  mapperclient.MapperClient
	authzFilePath string
}

func NewWatcher(mapperClient mapperclient.MapperClient, authzFilePath string) (*Watcher, error) {
	w := &Watcher{
		mu:            sync.Mutex{},
		seen:          SeenRecordsStore{},
		mapperClient:  mapperClient,
		authzFilePath: authzFilePath,
	}

	return w, nil
}

func (w *Watcher) processLogRecord(kafkaServer types.NamespacedName, record string) {
	authorizerRecord := AuthorizerRecord{
		Server: kafkaServer,
	}
	if err := AclAuthorizerRegex.MatchToTarget(record, &authorizerRecord); errors.Is(err, &regroup.NoMatchFoundError{}) {
		return
	} else if err != nil {
		logrus.Errorf("Error matching authorizer regex: %s", err)
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.seen[authorizerRecord] = time.Now()
}

func (w *Watcher) WatchForever(ctx context.Context, serverName types.NamespacedName, authzLogPath string) {
	t, err := tail.TailFile(authzLogPath, tail.Config{Follow: true, ReOpen: true})

	if err != nil {
		panic(err)
	}

	for line := range t.Lines {
		w.processLogRecord(serverName, line.Text)
	}
}

func (w *Watcher) Flush() SeenRecordsStore {
	w.mu.Lock()
	defer w.mu.Unlock()
	r := w.seen
	w.seen = SeenRecordsStore{}
	return r
}

func (w *Watcher) ReportResults(ctx context.Context) error {
	records := w.Flush()

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

	return w.mapperClient.ReportKafkaMapperResults(ctx, mapperclient.KafkaMapperResults{Results: results})
}

func (w *Watcher) RunForever(ctx context.Context, serverName types.NamespacedName) error {
	go w.WatchForever(ctx, serverName, w.authzFilePath)

	for {
		time.Sleep(viper.GetDuration(config.KafkaCooldownIntervalKey))
		if err := w.ReportResults(ctx); err != nil {
			logrus.WithError(err).Errorf("Failed reporting watcher results to mapper")
		}
	}
}
