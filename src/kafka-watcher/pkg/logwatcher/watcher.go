package logwatcher

import (
	"bufio"
	"context"
	"errors"
	"github.com/amit7itz/goset"
	"github.com/oriser/regroup"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"math"
	"path/filepath"
	"sync"
	"time"
)

// AclAuthorizerRegex matches & decodes AclAuthorizer log records.
// Sample log record for reference:
// [2023-03-12 13:51:55,904] INFO Principal = User:2.5.4.45=#13206331373734376636373865323137613636346130653335393130326638303662,CN=myclient.otterize-tutorial-kafka-mtls,O=SPIRE,C=US is Denied Operation = Describe from host = 10.244.0.27 on resource = Topic:LITERAL:mytopic for request = Metadata with resourceRefCount = 1 (kafka.authorizer.logger)
var AclAuthorizerRegex = regroup.MustCompile(
	`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d+\] [A-Z]+ Principal = User:\S+CN=(?P<serviceName>[a-z0-9-.]+)\.(?P<namespace>[a-z0-9-.]+),\S+ is (?P<access>\S+) Operation = (?P<operation>\S+) from host = (?P<host>\S+) on resource = Topic:LITERAL:(?P<topic>.+) for request = \S+ with resourceRefCount = \d+ \(kafka\.authorizer\.logger\)$`,
)

type AuthorizerRecord struct {
	ServiceName string `regroup:"serviceName"`
	Namespace   string `regroup:"namespace"`
	Access      string `regroup:"access"`
	Operation   string `regroup:"operation"`
	Host        string `regroup:"host"`
	Topic       string `regroup:"topic"`
}

type pod struct {
	name      string
	namespace string
}

type Watcher struct {
	clientset *kubernetes.Clientset
	mu        sync.Mutex
	seen      *goset.Set[AuthorizerRecord]
	pod       pod
}

func NewWatcher(name string, namespace string) (*Watcher, error) {
	// TODO: client from service account
	conf, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		return nil, err
	}

	cs, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		clientset: cs,
		mu:        sync.Mutex{},
		seen:      goset.NewSet[AuthorizerRecord](),
		pod:       pod{name: name, namespace: namespace},
	}

	return w, nil
}

func (w *Watcher) WatchOnce(ctx context.Context) error {
	logrus.Infof("Watching logs on pod %s.%s", w.pod.name, w.pod.namespace)
	podLogOpts := corev1.PodLogOptions{
		Follow:       true,
		SinceSeconds: lo.ToPtr(int64(math.Ceil(viper.GetDuration(config.CooldownIntervalKey).Seconds()))),
	}
	req := w.clientset.CoreV1().Pods(w.pod.namespace).GetLogs(w.pod.name, &podLogOpts)
	reader, err := req.Stream(ctx)
	if err != nil {
		return err
	}

	defer reader.Close()

	s := bufio.NewScanner(reader)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		r := AuthorizerRecord{}
		if err := AclAuthorizerRegex.MatchToTarget(s.Text(), &r); errors.Is(err, &regroup.NoMatchFoundError{}) {
			continue
		} else if err != nil {
			logrus.Errorf("Error matching authorizer regex: %s", err)
			continue
		}

		w.mu.Lock()
		w.seen.Add(r)
		w.mu.Unlock()
	}

	return nil
}

func (w *Watcher) WatchForever(ctx context.Context) {
	cooldownPeriod := viper.GetDuration(config.CooldownIntervalKey)
	for {
		err := w.WatchOnce(ctx)
		if err != nil {
			logrus.Errorf("Error watching pod logs: %s", err)
		}
		logrus.Infof("Watcher stopped, will retry after cooldown period (%s)...", cooldownPeriod)
		time.Sleep(cooldownPeriod)
	}
}

func (w *Watcher) Flush() []AuthorizerRecord {
	w.mu.Lock()
	r := w.seen.Items()
	w.seen = goset.NewSet[AuthorizerRecord]()
	w.mu.Unlock()
	return r
}

func (w *Watcher) RunForever(ctx context.Context) error {
	go w.WatchForever(ctx)

	for {
		select {
		case <-time.After(viper.GetDuration(config.ReportIntervalKey)):
			records := w.Flush()
			logrus.Infof("Reporting %d records", len(records))
		}
	}
}
