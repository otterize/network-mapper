package kubefinder

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
	"testing"
)

type KubeFinderTestSuite struct {
	suite.Suite
	testEnv          *envtest.Environment
	cfg              *rest.Config
	testNamespace    *corev1.Namespace
	mgrCtx           context.Context
	k8sClient        *kubernetes.Clientset
	mgrCtxCancelFunc context.CancelFunc
	mgr              manager.Manager
}

func (suite *KubeFinderTestSuite) SetupSuite() {
	suite.testEnv = &envtest.Environment{}
	var err error
	suite.cfg, err = suite.testEnv.Start()
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.cfg)

	suite.k8sClient, err = kubernetes.NewForConfig(suite.cfg)
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.k8sClient)
}

func (suite *KubeFinderTestSuite) TearDownSuite() {
	suite.Require().NoError(suite.testEnv.Stop())
}

// Runs before each test
func (suite *KubeFinderTestSuite) BeforeTest(suiteName, testName string) {
	suite.mgrCtx, suite.mgrCtxCancelFunc = context.WithCancel(context.Background())
	suite.testNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(fmt.Sprintf("test-%s-%s", suiteName, testName))},
	}
	var err error
	suite.testNamespace, err = suite.k8sClient.CoreV1().Namespaces().Create(context.Background(), suite.testNamespace, metav1.CreateOptions{})
	suite.Require().NoError(err)

	suite.mgr, err = manager.New(suite.cfg, manager.Options{MetricsBindAddress: "0"})
	suite.Require().NoError(err)
	go func() {
		err := suite.mgr.Start(suite.mgrCtx)
		suite.Require().NoError(err)
	}()
}

// Runs after each test
func (suite *KubeFinderTestSuite) AfterTest(suiteName, testName string) {
	suite.mgrCtxCancelFunc()
	err := suite.k8sClient.CoreV1().Namespaces().Delete(context.Background(), suite.testNamespace.Name, metav1.DeleteOptions{})
	suite.Require().NoError(err)
}

func (suite *KubeFinderTestSuite) addPod(name string, ip string) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.testNamespace.Name},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Name:            name,
				Image:           "nginx",
				ImagePullPolicy: "Always",
			},
		},
		},
	}
	err := suite.mgr.GetClient().Create(context.Background(), pod)
	suite.Require().NoError(err)

	if ip != "" {
		pod.Status.PodIP = ip
		pod.Status.PodIPs = []corev1.PodIP{{ip}}
		pod, err = suite.k8sClient.CoreV1().Pods(suite.testNamespace.Name).UpdateStatus(context.Background(), pod, metav1.UpdateOptions{})
		suite.Require().NoError(err)
	}
}

func (suite *KubeFinderTestSuite) TestResolveIpToPod() {
	kf, err := NewKubeFinder(suite.mgr)
	suite.Require().NoError(err)

	pod, err := kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().Nil(pod)
	suite.Require().Error(err)

	suite.addPod("some-pod", "2.2.2.2")
	suite.addPod("test-pod", "1.1.1.1")
	suite.addPod("pod-with-no-ip", "")

	suite.mgr.GetCache().WaitForCacheSync(context.Background())
	pod, err = kf.ResolveIpToPod(context.Background(), "1.1.1.1")
	suite.Require().NoError(err)
	suite.Require().Equal("test-pod", pod.Name)

}

func TestKubeFinderTestSuite(t *testing.T) {
	suite.Run(t, new(KubeFinderTestSuite))
}
