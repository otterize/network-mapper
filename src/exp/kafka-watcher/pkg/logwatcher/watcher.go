package logwatcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/oriser/regroup"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/exp/kafka-watcher/pkg/mapperclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
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
	clientset    *kubernetes.Clientset
	mu           sync.Mutex
	seen         SeenRecordsStore
	mapperClient mapperclient.MapperClient
	kafkaServers []types.NamespacedName
}

func NewWatcher(mapperClient mapperclient.MapperClient, kafkaServers []types.NamespacedName) (*Watcher, error) {
	conf, err := rest.InClusterConfig()

	if err != nil && !errors.Is(err, rest.ErrNotInCluster) {
		return nil, err
	}

	// We try building the REST Config from ./kube/config to support running the watcher locally
	if conf == nil {
		conf, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
		if err != nil {
			return nil, err
		}
	}

	cs, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		clientset:    cs,
		mu:           sync.Mutex{},
		seen:         SeenRecordsStore{},
		mapperClient: mapperClient,
		kafkaServers: kafkaServers,
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

func (w *Watcher) WatchOnce(ctx context.Context, kafkaServer types.NamespacedName, startTime time.Time) error {
	podLogOpts := corev1.PodLogOptions{
		SinceTime: &metav1.Time{Time: startTime},
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	req := w.clientset.CoreV1().Pods(kafkaServer.Namespace).GetLogs(kafkaServer.Name, &podLogOpts)
	reader, err := req.Stream(ctxTimeout)
	if err != nil {
		return err
	}

	defer reader.Close()

	s := bufio.NewScanner(reader)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		w.processLogRecord(kafkaServer, s.Text())
	}

	return nil
}

func (w *Watcher) WatchForever(ctx context.Context, kafkaServer types.NamespacedName) {
	log := logrus.WithField("pod", kafkaServer)
	cooldownPeriod := viper.GetDuration(config.KafkaCooldownIntervalKey)
	readFromTime := time.Now().Add(-(viper.GetDuration(config.KafkaCooldownIntervalKey)))
	for {
		log.Info("Watching logs")
		err := w.WatchOnce(ctx, kafkaServer, readFromTime)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			log.WithError(err).Error("Error watching logs")
		}
		readFromTime = time.Now()
		log.Infof("Waiting %s before watching logs again...", cooldownPeriod)
		time.Sleep(cooldownPeriod)
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
	logrus.Infof("Reporting %d records", len(records))

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

func (w *Watcher) RunForever(ctx context.Context) error {
	for _, kafkaServer := range w.kafkaServers {
		go w.WatchForever(ctx, kafkaServer)
	}

	for {
		time.Sleep(viper.GetDuration(config.KafkaCooldownIntervalKey))
		if err := w.ReportResults(ctx); err != nil {
			logrus.WithError(err).Errorf("Failed reporting watcher results to mapper")
		}
	}
}

func (w *Watcher) ValidateKafkaServers(ctx context.Context) error {
	for _, kafkaServer := range w.kafkaServers {
		p, err := w.clientset.CoreV1().Pods(kafkaServer.Namespace).Get(ctx, kafkaServer.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if p == nil {
			return fmt.Errorf("could not find kafka server pod: %s.%s", kafkaServer.Name, kafkaServer.Namespace)
		}
	}

	return nil
}
