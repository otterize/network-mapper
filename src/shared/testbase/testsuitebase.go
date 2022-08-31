package testbase

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"time"
)

type ControllerManagerTestSuiteBase struct {
	suite.Suite
	testEnv          *envtest.Environment
	cfg              *rest.Config
	TestNamespace    string
	K8sDirectClient  *kubernetes.Clientset
	mgrCtx           context.Context
	mgrCtxCancelFunc context.CancelFunc
	Mgr              manager.Manager
}

func (suite *ControllerManagerTestSuiteBase) SetupSuite() {
	suite.testEnv = &envtest.Environment{}
	var err error
	suite.cfg, err = suite.testEnv.Start()
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.cfg)

	suite.K8sDirectClient, err = kubernetes.NewForConfig(suite.cfg)
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.K8sDirectClient)
}

func (suite *ControllerManagerTestSuiteBase) TearDownSuite() {
	suite.Require().NoError(suite.testEnv.Stop())
}

func (suite *ControllerManagerTestSuiteBase) SetupTest() {
	suite.mgrCtx, suite.mgrCtxCancelFunc = context.WithCancel(context.Background())

	var err error
	suite.Mgr, err = manager.New(suite.cfg, manager.Options{MetricsBindAddress: "0"})
	suite.Require().NoError(err)
}

// BeforeTest happens AFTER the SetupTest()
func (suite *ControllerManagerTestSuiteBase) BeforeTest(_, testName string) {
	go func() {
		// We start the manager in "Before test" to allow operations that should happen before start to be run at SetupTest()
		err := suite.Mgr.Start(suite.mgrCtx)
		suite.Require().NoError(err)
	}()

	suite.TestNamespace = strings.ToLower(fmt.Sprintf("%s-%s", testName, time.Now().Format("20060102150405")))
	testNamespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: suite.TestNamespace},
	}
	var err error
	_, err = suite.K8sDirectClient.CoreV1().Namespaces().Create(context.Background(), testNamespaceObj, metav1.CreateOptions{})
	suite.Require().NoError(err)
}

func (suite *ControllerManagerTestSuiteBase) TearDownTest() {
	suite.mgrCtxCancelFunc()
	err := suite.K8sDirectClient.CoreV1().Namespaces().Delete(context.Background(), suite.TestNamespace, metav1.DeleteOptions{})
	suite.Require().NoError(err)
}

func (suite *ControllerManagerTestSuiteBase) AddPod(name string, podIp string, labels map[string]string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace, Labels: labels},
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

	if podIp != "" {
		pod.Status.PodIP = podIp
		pod.Status.PodIPs = []corev1.PodIP{{podIp}}
		pod, err = suite.K8sDirectClient.CoreV1().Pods(suite.TestNamespace).UpdateStatus(context.Background(), pod, metav1.UpdateOptions{})
		suite.Require().NoError(err)
	}
	return pod
}

func (suite *ControllerManagerTestSuiteBase) AddEndpoints(name string, podIps []string) *corev1.Endpoints {
	addresses := lo.Map(podIps, func(ip string, _ int) corev1.EndpointAddress {
		return corev1.EndpointAddress{IP: ip}
	})

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Subsets:    []corev1.EndpointSubset{{Addresses: addresses, Ports: []corev1.EndpointPort{{Name: "someport", Port: 8080, Protocol: corev1.ProtocolTCP}}}},
	}

	err := suite.Mgr.GetClient().Create(context.Background(), endpoints)
	suite.Require().NoError(err)
	return endpoints
}

func (suite *ControllerManagerTestSuiteBase) AddService(name string, podIps []string, selector map[string]string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Spec: corev1.ServiceSpec{Selector: selector,
			Ports: []corev1.ServicePort{{Name: "someport", Port: 8080, Protocol: corev1.ProtocolTCP}},
			Type:  corev1.ServiceTypeClusterIP,
		},
	}
	err := suite.Mgr.GetClient().Create(context.Background(), service)
	suite.Require().NoError(err)

	suite.AddEndpoints(name, podIps)
	return service
}

func (suite *ControllerManagerTestSuiteBase) AddReplicaSet(name string, podIps []string, podLabels map[string]string) *appsv1.ReplicaSet {
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: lo.ToPtr(int32(len(podIps))),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace, Labels: podLabels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{
						Name:            name,
						Image:           "nginx",
						ImagePullPolicy: "Always",
					},
				},
				},
			},
		},
	}
	err := suite.Mgr.GetClient().Create(context.Background(), replicaSet)
	suite.Require().NoError(err)

	for i, ip := range podIps {
		pod := suite.AddPod(fmt.Sprintf("%s-%d", name, i), ip, podLabels)
		pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion:         "apps/v1",
				Kind:               "ReplicaSet",
				BlockOwnerDeletion: lo.ToPtr(true),
				Controller:         lo.ToPtr(true),
				Name:               replicaSet.Name,
				UID:                replicaSet.UID,
			},
		}
		err := suite.Mgr.GetClient().Update(context.Background(), pod)
		suite.Require().NoError(err)
	}

	return replicaSet
}

func (suite *ControllerManagerTestSuiteBase) AddDeployment(name string, podIps []string, podLabels map[string]string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: lo.ToPtr(int32(len(podIps))),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: suite.TestNamespace, Labels: podLabels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{
					{
						Name:            name,
						Image:           "nginx",
						ImagePullPolicy: "Always",
					},
				},
				},
			},
		},
	}
	err := suite.Mgr.GetClient().Create(context.Background(), deployment)
	suite.Require().NoError(err)

	replicaSet := suite.AddReplicaSet(name, podIps, podLabels)
	replicaSet.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			BlockOwnerDeletion: lo.ToPtr(true),
			Controller:         lo.ToPtr(true),
			Name:               deployment.Name,
			UID:                deployment.UID,
		},
	}
	err = suite.Mgr.GetClient().Update(context.Background(), replicaSet)
	suite.Require().NoError(err)

	return deployment
}

func (suite *ControllerManagerTestSuiteBase) AddDeploymentWithService(name string, podIps []string, podLabels map[string]string) (*appsv1.Deployment, *corev1.Service) {
	deployment := suite.AddDeployment(name, podIps, podLabels)
	service := suite.AddService(name, podIps, podLabels)
	return deployment, service
}
