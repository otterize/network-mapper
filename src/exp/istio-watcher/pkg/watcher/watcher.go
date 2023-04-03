package istiowatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/oriser/regroup"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/config"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
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
	IstioProxyTotalRequestsCMD = "pilot-agent request GET stats?format=json&filter=istio_requests_total"
	IstioSidecarContainerName  = "istio-proxy"
	IstioPodsLabelSelector     = "security.istio.io/tlsMode"
	MetricsBufferedChannelSize = 100
)

var (
	EnvoyConnectionMetricRegex = regroup.MustCompile(`.*(?P<source_workload>source_workload\.\b[^.]+).*(?P<source_namespace>source_workload_namespace\.\b[^.]+).*(?P<destination_workload>destination_workload\.\b[^.]+).*(?P<destination_namespace>destination_workload_namespace\.\b[^.]+).*(?P<request_path>request_path\.[^.]+)`)
)

var (
	ConnectionInfoInsufficient = errors.New("connection info partial or empty")
)

type IstioWatcher struct {
	clientset    *kubernetes.Clientset
	config       *rest.Config
	mapperClient mapperclient.MapperClient
	connections  map[*ConnectionWithPath]time.Time
}

type ConnectionWithPath struct {
	SourceWorkload       string `regroup:"source_workload"`
	SourceNamespace      string `regroup:"source_namespace"`
	DestinationWorkload  string `regroup:"destination_workload"`
	DestinationNamespace string `regroup:"destination_namespace"`
	RequestPath          string `regroup:"request_path"`
}

func (p *ConnectionWithPath) hasMissingInfo() bool {
	for _, field := range []string{p.SourceWorkload, p.SourceNamespace, p.DestinationWorkload, p.DestinationNamespace} {
		if field == "" || strings.Contains(field, "unknown") {
			return true
		}
	}
	if p.RequestPath == "" {
		return true
	}

	return false
}

// omitMetricsFieldsFromConnection drops the metric name and uses the value alone in the connection
// Since we cant use lookaheads in our regex matching, connections fields are parsed with their metric name as well
// e.g. for source workload we get "source_workload.some-client", and we need to parse "some-client" and remove the metric name
func (p *ConnectionWithPath) omitMetricsFieldsFromConnection() {
	p.SourceWorkload = strings.Split(p.SourceWorkload, ".")[1]
	p.DestinationWorkload = strings.Split(p.DestinationWorkload, ".")[1]
	p.SourceNamespace = strings.Split(p.SourceNamespace, ".")[1]
	p.DestinationNamespace = strings.Split(p.DestinationNamespace, ".")[1]
	p.RequestPath = strings.Split(p.RequestPath, ".")[1]
}

type EnvoyMetrics struct {
	Stats []Metric `json:"stats"`
}

type Metric struct {
	Name string `json:"name"`
}

func NewWatcher(mapperClient mapperclient.MapperClient) (*IstioWatcher, error) {
	//conf, err := rest.InClusterConfig()
	//if err != nil {
	//	return nil, err
	//}
	conf, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	m := &IstioWatcher{
		clientset:    clientset,
		config:       conf,
		mapperClient: mapperClient,
		connections:  map[*ConnectionWithPath]time.Time{},
	}

	return m, nil
}

func (m *IstioWatcher) Flush() map[*ConnectionWithPath]time.Time {
	r := m.connections
	m.connections = map[*ConnectionWithPath]time.Time{}
	return r
}

func (m *IstioWatcher) CollectIstioConnectionMetrics(ctx context.Context, namespace string) error {
	sendersErrGroup, sendersCtx := errgroup.WithContext(ctx)
	receiversErrGroup, _ := errgroup.WithContext(ctx)
	metricsChan := make(chan *EnvoyMetrics, MetricsBufferedChannelSize)

	podList, err := m.clientset.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{LabelSelector: IstioPodsLabelSelector})
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		// Known for loop gotcha with goroutines
		curr := pod
		sendersErrGroup.Go(func() error {
			if err := m.getEnvoyMetricsFromSidecar(sendersCtx, curr, metricsChan); err != nil {
				logrus.Errorf("Failed fetching request metrics from pod %s", curr.Name)
				return err
			}
			return nil
		})
	}
	receiversErrGroup.Go(func() error {
		// Function call below updates a map which isn't concurrent-safe.
		// Needs to be taken into consideration if the code should ever change to use multiple goroutines
		if err := m.convertMetricsToConnections(sendersCtx, metricsChan); err != nil {
			return err
		}
		return nil
	})

	if err := sendersErrGroup.Wait(); err != nil {
		return err
	}
	sendersCtx.Done()

	if err := receiversErrGroup.Wait(); err != nil {
		return err
	}

	close(metricsChan)
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
		return err
	}

	var outBuf bytes.Buffer
	streamOpts := remotecommand.StreamOptions{Stdout: &outBuf}
	err = exec.StreamWithContext(ctx, streamOpts)
	if err != nil {
		return err
	}

	metrics := &EnvoyMetrics{}
	if err := json.NewDecoder(&outBuf).Decode(metrics); err != nil {
		return err
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

func (m *IstioWatcher) convertMetricsToConnections(sendersCtx context.Context, metricsChan <-chan *EnvoyMetrics) error {
	for {
		select {
		case metrics := <-metricsChan:
			for _, metric := range metrics.Stats {
				conn, err := m.buildConnectionFromMetric(metric)
				if err != nil && errors.Is(err, ConnectionInfoInsufficient) {
					continue
				}
				if err != nil {
					return err
				}
				m.connections[conn] = time.Now()
			}
		case <-sendersCtx.Done():
			logrus.Debugln("Got done signal")
			return nil
		}
	}
}

func (m *IstioWatcher) buildConnectionFromMetric(metric Metric) (*ConnectionWithPath, error) {
	conn := &ConnectionWithPath{}
	err := EnvoyConnectionMetricRegex.MatchToTarget(metric.Name, conn)
	if err != nil && errors.Is(err, &regroup.NoMatchFoundError{}) {
		return nil, ConnectionInfoInsufficient
	}
	if err != nil {
		return nil, err
	}
	if conn.hasMissingInfo() {
		return nil, ConnectionInfoInsufficient
	}

	conn.omitMetricsFieldsFromConnection()
	return conn, nil
}

func (m *IstioWatcher) ReportResults(ctx context.Context) {
	for {
		time.Sleep(viper.GetDuration(sharedconfig.ReportIntervalKey))
		connections := m.Flush()
		if len(connections) == 0 {
			continue
		}

		logrus.Infof("Reporting %d connections", len(connections))
		results := ToGraphQLIstioConnections(connections)
		if err := m.mapperClient.ReportIstioConnections(ctx, mapperclient.IstioConnectionResults{Results: results}); err != nil {
			logrus.WithError(err).Errorf("Failed reporting Istio connection results to mapper")
		}
	}
}

func (m *IstioWatcher) RunForever(ctx context.Context) error {
	go m.ReportResults(ctx)
	cooldownPeriod := viper.GetDuration(sharedconfig.CooldownIntervalKey)
	for {
		logrus.Info("Retrieving 'istio_total_requests' metric from Istio sidecars")
		if err := m.CollectIstioConnectionMetrics(ctx, viper.GetString(config.NamespaceKey)); err != nil {
			logrus.WithError(err).Debugf("Failed getting connection metrics from Istio sidecars")
		}
		logrus.Infof("Istio mapping stopped, will retry after cool down period (%s)...", cooldownPeriod)
		time.Sleep(cooldownPeriod)
	}

}
