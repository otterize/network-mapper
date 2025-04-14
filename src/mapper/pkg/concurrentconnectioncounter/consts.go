package concurrentconnectioncounter

const (
	SocketScanServiceIntentResolution string = "addSocketScanServiceIntent"
	SocketScanPodIntentResolution     string = "addSocketScanPodIntent"
	TCPTrafficIntentResolution        string = "handleInternalTrafficTCPResult"
	DNSTrafficIntentResolution        string = "handleDNSCaptureResultsAsKubernetesPods"
	KafkaResultIntentResolution       string = "handleReportKafkaMapperResults"
	IstioResultIntentResolution       string = "handleReportIstioConnectionResults"
)
