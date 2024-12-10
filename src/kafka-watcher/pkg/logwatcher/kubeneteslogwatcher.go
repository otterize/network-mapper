package logwatcher

import (
	"bufio"
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/mapperclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"reflect"
	"sync"
	"time"
)

type KubernetesLogWatcher struct {
	baseWatcher
	clientset    *kubernetes.Clientset
	kafkaServers []types.NamespacedName
}

func NewKubernetesLogWatcher(mapperClient *mapperclient.Client, kafkaServers []types.NamespacedName) (*KubernetesLogWatcher, error) {
	conf, err := rest.InClusterConfig()

	if err != nil && !errors.Is(err, rest.ErrNotInCluster) {
		return nil, errors.Wrap(err)
	}

	// We try building the REST Config from ./kube/config to support running the watcher locally
	if conf == nil {
		conf, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	cs, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	w := &KubernetesLogWatcher{
		baseWatcher: baseWatcher{
			mu:           sync.Mutex{},
			seen:         SeenRecordsStore{},
			mapperClient: mapperClient,
		},
		clientset:    cs,
		kafkaServers: kafkaServers,
	}

	return w, nil
}

func (w *KubernetesLogWatcher) RunForever(ctx context.Context) error {
	err := w.validateKafkaServers(ctx)

	if err != nil {
		return errors.Wrap(err)
	}

	for _, kafkaServer := range w.kafkaServers {
		go w.watchForever(ctx, kafkaServer)
	}

	for {
		time.Sleep(viper.GetDuration(config.KafkaReportIntervalKey))
		if err := w.reportResults(ctx); err != nil {
			logrus.WithError(err).Errorf("Failed reporting watcher results to mapper")
		}
	}
}

func (w *KubernetesLogWatcher) watchOnce(ctx context.Context, kafkaServer types.NamespacedName, startTime time.Time) error {
	pod, err := w.clientset.CoreV1().Pods(kafkaServer.Namespace).Get(ctx, kafkaServer.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err)
	}
	if pod.Status.Phase != corev1.PodRunning {
		logrus.Debugf("Kafka server %s is not running, skipping logs for this iteration", kafkaServer.String())
		return nil
	}
	podLogOpts := corev1.PodLogOptions{
		SinceTime: &metav1.Time{Time: startTime},
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	req := w.clientset.CoreV1().Pods(kafkaServer.Namespace).GetLogs(kafkaServer.Name, &podLogOpts)
	reader, err := req.Stream(ctxTimeout)
	if err != nil {
		return errors.Wrap(err)
	}

	defer reader.Close()

	s := bufio.NewScanner(reader)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		w.processLogRecord(kafkaServer, s.Text())
	}

	return nil
}

func (w *KubernetesLogWatcher) watchForever(ctx context.Context, kafkaServer types.NamespacedName) {
	log := logrus.WithField("pod", kafkaServer)
	cooldownPeriod := viper.GetDuration(config.KafkaCooldownIntervalKey)
	readFromTime := time.Now().Add(-(viper.GetDuration(config.KafkaCooldownIntervalKey)))

	for {
		log.Info("Watching logs")
		err := w.watchOnce(ctx, kafkaServer, readFromTime)

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

func (w *KubernetesLogWatcher) validateKafkaServers(ctx context.Context) error {
	invalidServers := make([]string, 0)
	for _, kafkaServer := range w.kafkaServers {
		_, err := w.clientset.CoreV1().Pods(kafkaServer.Namespace).Get(ctx, kafkaServer.Name, metav1.GetOptions{})
		if err != nil {
			invalidServers = append(invalidServers, kafkaServer.String())
		}
	}
	if len(invalidServers) == 0 {
		return nil
	}
	logrus.Errorf("The following Kafka servers were not found or unreachable: %s", invalidServers)

	if reflect.DeepEqual(invalidServers, w.kafkaServers) {
		return errors.New("failed validating all Kafka servers")
	}
	validServers := make([]string, 0)
	for _, server := range w.kafkaServers {
		if !slices.Contains(invalidServers, server.String()) {
			validServers = append(validServers, server.String())
		}
	}

	logrus.Infof("Kafka watcher will run for the following servers: %s", validServers)
	return nil
}
