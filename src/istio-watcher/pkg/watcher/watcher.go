package istiowatcher

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"strings"
	"time"
)

/*
Envoy metric sample
{
"name": "istiocustom.istio_requests_total.reporter.source.source_workload.sleep.source_canonical_service.sleep

			.source_canonical_revision.latest.source_workload_namespace.default.source_principal.unknown.source_app
			.sleep.source_version.source_cluster.Kubernetes.destination_workload.unknown.destination_workload_namespace
			.unknown.destination_principal.unknown.destination_app.unknown.destination_version.unknown.destination_service
			.security.ubuntu.com.destination_canonical_service.unknown.destination_canonical_revision.latest
			.destination_service_name.PassthroughCluster.destination_service_namespace.unknown.destination_cluster
			.unknown.request_protocol.http.response_code.200.grpc_response_status.response_flags.-
			.connection_security_policy.unknown.request_path./ubuntu/dists/jammy-security/InRelease"
	}
*/
const (
	IstioProxyTotalRequestsCMD = "pilot-agent request GET stats?format=json&filter=istio_requests_total\\.*reporter\\.source"
	IstioSidecarContainerName  = "istio-proxy"
	IstioPodsLabelSelector     = "security.istio.io/tlsMode"
	MetricsBufferedChannelSize = 100
)

var (
	ConnectionInfoInsufficient = errors.New("connection info partial or empty")
	GroupNames                 = []string{
		"source_workload",
		"source_workload_namespace",
		"destination_workload",
		"destination_workload_namespace",
		"request_method",
		"request_path",
	}
)

type ConnectionWithPath struct {
	SourceWorkload       string `json:"source_workload"`
	SourceNamespace      string `json:"source_workload_namespace"`
	DestinationWorkload  string `json:"destination_workload"`
	DestinationNamespace string `json:"destination_workload_namespace"`
	RequestPath          string `json:"request_path"`
	RequestMethod        string `json:"request_method"`
}

type IstioWatcher struct {
	clientset    *kubernetes.Clientset
	config       *rest.Config
	reporter     IstioReporter
	connections  map[ConnectionWithPath]time.Time
	metricsCount map[string]int
}

func (p *ConnectionWithPath) hasMissingInfo() bool {
	for _, field := range []string{p.SourceWorkload, p.SourceNamespace, p.DestinationWorkload, p.DestinationNamespace} {
		if field == "" || strings.Contains(field, "unknown") {
			return true
		}
	}

	return p.RequestPath == "" || p.RequestPath == "unknown"
}

type EnvoyMetrics struct {
	Stats []Metric `json:"stats"`
}

type Metric struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type IstioReporter interface {
	ReportIstioConnectionResults(ctx context.Context, results model.IstioConnectionResults) (bool, error)
}

// NewWatcher The Istio watcher uses this interface because it used to be a standalone component that communicates with the network mapper, and was then integrated into the network mapper.
func NewWatcher(resolver IstioReporter) (*IstioWatcher, error) {
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

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	m := &IstioWatcher{
		clientset:    clientset,
		config:       conf,
		reporter:     resolver,
		connections:  map[ConnectionWithPath]time.Time{},
		metricsCount: map[string]int{},
	}

	return m, nil
}

func (m *IstioWatcher) Flush() map[ConnectionWithPath]time.Time {
	r := m.connections
	m.connections = map[ConnectionWithPath]time.Time{}
	return r
}

func (m *IstioWatcher) CollectIstioConnectionMetrics(ctx context.Context, namespace string) error {
	sendersErrGroup, sendersCtx := errgroup.WithContext(ctx)
	sendersErrGroup.SetLimit(10)

	receiverErrGroup, _ := errgroup.WithContext(ctx)
	metricsChan := make(chan *EnvoyMetrics, MetricsBufferedChannelSize)

	podList, err := m.clientset.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{LabelSelector: IstioPodsLabelSelector})
	if err != nil {
		return errors.Wrap(err)
	}

	for _, pod := range podList.Items {
		// Known for loop gotcha with goroutines
		curr := pod
		sendersErrGroup.Go(func() error {
			if err := m.getEnvoyMetricsFromSidecar(sendersCtx, curr, metricsChan); err != nil {
				logrus.WithError(err).Errorf("Failed fetching request metrics from pod %s", curr.Name)
				return nil // Intentionally logging error and returning nil to not cancel err group context
			}
			return nil
		})
	}
	receiverErrGroup.Go(func() error {
		// Function call below updates a map which isn't concurrent-safe.
		// Needs to be taken into consideration if the code should ever change to use multiple goroutines
		if err := m.convertMetricsToConnections(metricsChan); err != nil {
			return errors.Wrap(err)
		}
		return nil
	})

	if err := sendersErrGroup.Wait(); err != nil {
		return errors.Wrap(err)
	}
	close(metricsChan)

	if err := receiverErrGroup.Wait(); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (m *IstioWatcher) getEnvoyMetricsFromSidecar(ctx context.Context, pod corev1.Pod, metricsChan chan<- *EnvoyMetrics) error {
	req := m.clientset.CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   strings.Split(IstioProxyTotalRequestsCMD, " "),
			Stdout:    true, // We omit stderr and we error according to return code from executed cmd
			Container: IstioSidecarContainerName,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(m.config, "POST", req.URL())
	if err != nil {
		return errors.Wrap(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, viper.GetDuration(config.MetricFetchTimeoutKey))
	defer cancel()

	var outBuf bytes.Buffer
	streamOpts := remotecommand.StreamOptions{Stdout: &outBuf}
	err = exec.StreamWithContext(timeoutCtx, streamOpts)
	if err != nil {
		return errors.Wrap(err)
	}

	metrics := &EnvoyMetrics{}
	if err := json.NewDecoder(&outBuf).Decode(metrics); err != nil {
		return errors.Wrap(err)
	}
	metricsWithPath := make([]Metric, 0)
	for _, metric := range metrics.Stats {
		if strings.Contains(metric.Name, "request_path") {
			metricsWithPath = append(metricsWithPath, metric)
		}
	}

	metrics.Stats = metricsWithPath
	if len(metrics.Stats) == 0 {
		return nil
	}

	metricsChan <- metrics
	return nil
}

func (m *IstioWatcher) convertMetricsToConnections(metricsChan <-chan *EnvoyMetrics) error {
	for {
		metrics, more := <-metricsChan
		if !more {
			return nil
		}

		for _, metric := range metrics.Stats {
			if !m.isMetricNew(metric) {
				continue
			}

			conn, err := m.buildConnectionFromMetric(metric)
			if err != nil && errors.Is(err, ConnectionInfoInsufficient) {
				continue
			}
			if err != nil {
				return errors.Wrap(err)
			}
			m.connections[conn] = time.Now()
		}
	}
}

func hashString(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (m *IstioWatcher) isMetricNew(metric Metric) bool {
	// Metrics can be 1500 characters long the hash is solely for optimization purposes
	key := hashString(metric.Name)
	previousCount, found := m.metricsCount[key]
	if !found || previousCount != metric.Value {
		m.metricsCount[key] = metric.Value
		return true
	}

	return false
}

func extractRegexGroups(inputString string, groupNames []string) (*ConnectionWithPath, error) {
	connection := &ConnectionWithPath{}
	for _, groupName := range groupNames {
		groupKey := groupName + "."
		groupIndex := strings.Index(inputString, groupKey)
		if groupIndex == -1 {
			continue
		}
		groupValue := inputString[groupIndex+len(groupKey):]
		if strings.IndexByte(groupValue, '.') != -1 {
			groupValue = groupValue[:strings.IndexByte(groupValue, '.')]
		}

		switch groupName {
		case "source_workload":
			connection.SourceWorkload = groupValue
		case "source_workload_namespace":
			connection.SourceNamespace = groupValue
		case "destination_workload":
			connection.DestinationWorkload = groupValue
		case "destination_workload_namespace":
			connection.DestinationNamespace = groupValue
		case "request_path":
			connection.RequestPath = groupValue
		case "request_method":
			connection.RequestMethod = groupValue
		default:
			return nil, errors.Errorf("unknown group name: %s", groupName)
		}
	}
	return connection, nil
}

func (m *IstioWatcher) buildConnectionFromMetric(metric Metric) (ConnectionWithPath, error) {
	conn, err := extractRegexGroups(metric.Name, GroupNames)

	if err != nil {
		return ConnectionWithPath{}, errors.Wrap(err)
	}

	if conn.hasMissingInfo() {
		return ConnectionWithPath{}, ConnectionInfoInsufficient
	}

	return *conn, nil
}

func (m *IstioWatcher) ReportResults(ctx context.Context) {
	for {
		time.Sleep(viper.GetDuration(config.IstioReportIntervalKey))
		err := m.reportResults(ctx)
		if err != nil {
			logrus.WithError(err).Errorf("Failed reporting Istio connection results to mapper")
		}
	}
}

func (m *IstioWatcher) reportResults(ctx context.Context) error {
	connections := m.Flush()
	if len(connections) == 0 {
		logrus.Debugln("No connections found in metrics - skipping report")
		return nil
	}

	logrus.Debugf("Reporting %d connections", len(connections))
	results := ToGraphQLIstioConnections(connections)
	_, err := m.reporter.ReportIstioConnectionResults(ctx, model.IstioConnectionResults{Results: results})
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func (m *IstioWatcher) RunForever(ctx context.Context) error {
	go m.ReportResults(ctx)
	cooldownPeriod := viper.GetDuration(config.IstioCooldownIntervalKey)
	for {
		logrus.Debug("Retrieving 'istio_total_requests' metric from Istio sidecars")
		if err := m.CollectIstioConnectionMetrics(ctx, viper.GetString(config.IstioRestrictCollectionToNamespace)); err != nil {
			logrus.WithError(err).Debugf("Failed getting connection metrics from Istio sidecars")
		}
		logrus.Debugf("Istio mapping stopped, will retry after cool down period (%s)...", cooldownPeriod)
		time.Sleep(cooldownPeriod)
	}
}
