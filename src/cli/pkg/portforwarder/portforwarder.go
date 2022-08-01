package portforwarder

import (
	"context"
	"fmt"
	"io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type PortForwarder struct {
	namespace   string
	serviceName string
	servicePort int
}

func NewPortForwarder(namespace string, serviceName string, servicePort int) *PortForwarder {
	return &PortForwarder{
		namespace:   namespace,
		serviceName: serviceName,
		servicePort: servicePort,
	}
}

func (p *PortForwarder) Start(ctx context.Context) (localPort int, err error) {
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		return 0, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return 0, err
	}
	srv, err := clientSet.CoreV1().Services(p.namespace).Get(context.TODO(), p.serviceName, v1.GetOptions{})
	if err != nil {
		return 0, err
	}
	podList, err := clientSet.CoreV1().Pods(p.namespace).List(context.TODO(), v1.ListOptions{LabelSelector: labels.SelectorFromSet(srv.Spec.Selector).String()})
	if err != nil {
		return 0, err
	}
	if len(podList.Items) == 0 {
		return 0, fmt.Errorf("service %s has no pods", p.serviceName)
	}
	mapperPod := podList.Items[0]
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		p.namespace, mapperPod.Name)
	hostIP := strings.TrimLeft(config.Host, "https://")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return 0, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	readyChan := make(chan struct{})

	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", 0, p.servicePort)}, ctx.Done(), readyChan, io.Discard, os.Stderr)
	if err != nil {
		return 0, err
	}
	go func() {
		err = fw.ForwardPorts()
		if err != nil {
			panic(err)
		}
	}()
	select {
	case <-readyChan:
		break
	}
	ports, err := fw.GetPorts()
	if err != nil {
		return 0, err
	}
	localPort = int(ports[0].Local)
	return localPort, nil
}
