package testutils

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"time"
)

type ManagerTestSuite struct {
	suite.Suite
	testEnv          *envtest.Environment
	cfg              *rest.Config
	TestNamespace    string
	K8sDirectClient  *kubernetes.Clientset
	mgrCtx           context.Context
	mgrCtxCancelFunc context.CancelFunc
	Mgr              manager.Manager
}

func (suite *ManagerTestSuite) SetupSuite() {
	suite.testEnv = &envtest.Environment{}
	var err error
	suite.cfg, err = suite.testEnv.Start()
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.cfg)

	suite.K8sDirectClient, err = kubernetes.NewForConfig(suite.cfg)
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.K8sDirectClient)
}

func (suite *ManagerTestSuite) TearDownSuite() {
	suite.Require().NoError(suite.testEnv.Stop())
}

func (suite *ManagerTestSuite) BeforeTest(suiteName, testName string) {
	suite.mgrCtx, suite.mgrCtxCancelFunc = context.WithCancel(context.Background())
	suite.TestNamespace = strings.ToLower(fmt.Sprintf("test-%s-%s-%s", suiteName, testName, time.Now().Format("20060102150405")))
	testNamespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: suite.TestNamespace},
	}
	var err error
	_, err = suite.K8sDirectClient.CoreV1().Namespaces().Create(context.Background(), testNamespaceObj, metav1.CreateOptions{})
	suite.Require().NoError(err)

	suite.Mgr, err = manager.New(suite.cfg, manager.Options{MetricsBindAddress: "0"})
	suite.Require().NoError(err)
	go func() {
		err := suite.Mgr.Start(suite.mgrCtx)
		suite.Require().NoError(err)
	}()
}

func (suite *ManagerTestSuite) AfterTest(suiteName, testName string) {
	suite.mgrCtxCancelFunc()
	err := suite.K8sDirectClient.CoreV1().Namespaces().Delete(context.Background(), suite.TestNamespace, metav1.DeleteOptions{})
	suite.Require().NoError(err)
}

func (suite *ManagerTestSuite) AddPod(name string, ip string) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Name:            name,
				Image:           "nginx",
				ImagePullPolicy: "Always",
			},
		},
		},
	}
	err := suite.Mgr.GetClient().Create(context.Background(), pod)
	suite.Require().NoError(err)

	if ip != "" {
		pod.Status.PodIP = ip
		pod.Status.PodIPs = []corev1.PodIP{{ip}}
		pod, err = suite.K8sDirectClient.CoreV1().Pods(suite.TestNamespace).UpdateStatus(context.Background(), pod, metav1.UpdateOptions{})
		suite.Require().NoError(err)
	}
}

func (suite *ManagerTestSuite) AddEndpoints(name string, ips []string) {
	addresses := make([]corev1.EndpointAddress, 0, len(ips))
	for _, ip := range ips {
		addresses = append(addresses, corev1.EndpointAddress{IP: ip})
	}

	var endpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Subsets:    []corev1.EndpointSubset{{Addresses: addresses, Ports: []corev1.EndpointPort{{Name: "someport", Port: 8080, Protocol: corev1.ProtocolTCP}}}},
	}

	err := suite.Mgr.GetClient().Create(context.Background(), endpoints)
	suite.Require().NoError(err)
}

func (suite *ManagerTestSuite) AddService(name string) {
	pod := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Spec: corev1.ServiceSpec{Selector: map[string]string{},
			Ports: []corev1.ServicePort{{Name: "someport", Port: 8080, Protocol: corev1.ProtocolTCP}},
		},
	}
	err := suite.Mgr.GetClient().Create(context.Background(), pod)
	suite.Require().NoError(err)
}
