package testbase

import (
	"context"
	"fmt"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"strings"
	"time"
)

const waitForCreationInterval = 200 * time.Millisecond
const waitForCreationTimeout = 3 * time.Second

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

func (s *ControllerManagerTestSuiteBase) SetupTest() {
	s.testEnv = &envtest.Environment{}
	var err error
	s.cfg, err = s.testEnv.Start()
	s.Require().NoError(err)
	s.Require().NotNil(s.cfg)
	logrus.SetLevel(logrus.DebugLevel)

	s.K8sDirectClient, err = kubernetes.NewForConfig(s.cfg)
	s.Require().NoError(err)
	s.Require().NotNil(s.K8sDirectClient)
	s.mgrCtx, s.mgrCtxCancelFunc = context.WithCancel(context.Background())

	s.Mgr, err = manager.New(s.cfg, manager.Options{Metrics: server.Options{BindAddress: "0"}})
	s.Require().NoError(err)
	testName := s.T().Name()[strings.LastIndex(s.T().Name(), "/")+1:]
	s.TestNamespace = strings.ToLower(fmt.Sprintf("%s-%s", testName, time.Now().Format("20060102150405")))
	testNamespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.TestNamespace},
	}
	_, err = s.K8sDirectClient.CoreV1().Namespaces().Create(context.Background(), testNamespaceObj, metav1.CreateOptions{})
	s.Require().NoError(err)
}

// BeforeTest happens AFTER the SetupTest()
func (s *ControllerManagerTestSuiteBase) BeforeTest(_, testName string) {
	go func() {
		// We start the manager in "Before test" to allow operations that should happen before start to be run at SetupTest()
		err := s.Mgr.Start(s.mgrCtx)
		s.Require().NoError(err)
	}()
	s.Mgr.GetCache().WaitForCacheSync(context.Background()) // waiting for manager to start
}

func (s *ControllerManagerTestSuiteBase) TearDownTest() {
	s.mgrCtxCancelFunc()
	err := s.K8sDirectClient.CoreV1().Namespaces().Delete(context.Background(), s.TestNamespace, metav1.DeleteOptions{})
	s.Require().NoError(err)
	s.Require().NoError(s.testEnv.Stop())
}

// waitForObjectToBeCreated tries to get an object multiple times until it is available in the k8s API server
func (s *ControllerManagerTestSuiteBase) waitForObjectToBeCreated(obj client.Object) {
	s.Require().NoError(wait.PollUntilContextTimeout(context.Background(),
		waitForCreationInterval,
		waitForCreationTimeout,
		true,
		func(ctx context.Context) (done bool, err error) {
			err = s.Mgr.GetClient().Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, errors.Wrap(err)
			}
			return true, nil
		}),
	)
}

func (s *ControllerManagerTestSuiteBase) AddPod(name string, podIp string, labels map[string]string, ownerRefs []metav1.OwnerReference) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.TestNamespace, Labels: labels},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Name:            name,
				Image:           "nginx",
				ImagePullPolicy: "Always",
			},
		},
		},
	}

	if len(ownerRefs) != 0 {
		pod.SetOwnerReferences(ownerRefs)
	}

	err := s.Mgr.GetClient().Create(context.Background(), pod)
	s.Require().NoError(err)

	// Prevents race - UpdateStatus can alter the pod.
	podCopy := pod.DeepCopy()
	if podIp != "" {
		pod.Status.PodIP = podIp
		pod.Status.PodIPs = []corev1.PodIP{{IP: podIp}}
		pod.Status.Phase = corev1.PodRunning
		pod.Status.DeepCopyInto(&podCopy.Status)
		_, err = s.K8sDirectClient.CoreV1().Pods(s.TestNamespace).UpdateStatus(context.Background(), pod, metav1.UpdateOptions{})
		s.Require().NoError(err)
	}
	s.waitForObjectToBeCreated(pod)
	return podCopy
}

func (s *ControllerManagerTestSuiteBase) AddEndpoints(name string, pods []*corev1.Pod, port *int) *corev1.Endpoints {
	addresses := lo.Map(pods, func(pod *corev1.Pod, _ int) corev1.EndpointAddress {
		return corev1.EndpointAddress{IP: pod.Status.PodIP, TargetRef: &corev1.ObjectReference{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace}}
	})

	endpointPort := 8080
	if port != nil {
		endpointPort = *port
	}
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("svc-%s", name), Namespace: s.TestNamespace},
		Subsets:    []corev1.EndpointSubset{{Addresses: addresses, Ports: []corev1.EndpointPort{{Name: "someport", Port: int32(endpointPort), Protocol: corev1.ProtocolTCP}}}},
	}

	s.Require().NotEmpty(addresses[0].IP)
	err := s.Mgr.GetClient().Create(context.Background(), endpoints)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(endpoints)
	return endpoints
}

func (s *ControllerManagerTestSuiteBase) AddService(name string, selector map[string]string, serviceIp string, pods []*corev1.Pod) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("svc-%s", name), Namespace: s.TestNamespace},
		Spec: corev1.ServiceSpec{Selector: selector,
			Ports:      []corev1.ServicePort{{Name: "someport", Port: 8080, Protocol: corev1.ProtocolTCP}},
			Type:       corev1.ServiceTypeClusterIP,
			ClusterIP:  serviceIp,
			ClusterIPs: []string{serviceIp},
		},
	}
	err := s.Mgr.GetClient().Create(context.Background(), service)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(service)

	s.AddEndpoints(name, pods, nil)
	return service
}

func (s *ControllerManagerTestSuiteBase) AddServiceWithIngress(name string, selector map[string]string, serviceIp string, externalIP string, pods []*corev1.Pod, port int) *corev1.Service {
	serviceName := fmt.Sprintf("svc-%s", name)
	status := corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{{IP: externalIP}},
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: s.TestNamespace},
		Spec: corev1.ServiceSpec{Selector: selector,
			Ports:      []corev1.ServicePort{{Name: "someport", Port: int32(port), Protocol: corev1.ProtocolTCP}},
			Type:       corev1.ServiceTypeLoadBalancer,
			ClusterIP:  serviceIp,
			ClusterIPs: []string{serviceIp},
		},
	}
	err := s.Mgr.GetClient().Create(context.Background(), service)
	s.Require().NoError(err)

	service.Status = status
	err = s.Mgr.GetClient().Status().Update(context.Background(), service)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(service)
	s.AddIngress(name, serviceName, s.TestNamespace, externalIP, port)
	s.AddEndpoints(name, pods, &port)
	return service
}

func (s *ControllerManagerTestSuiteBase) AddIngress(name string, serviceName string, serviceNamespace, externalIP string, port int) *networkingv1.Ingress {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ingress-%s", name), Namespace: s.TestNamespace},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, serviceNamespace),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: lo.ToPtr(networkingv1.PathTypeExact),
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: int32(port),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{{IP: externalIP}},
			},
		},
	}

	err := s.Mgr.GetClient().Create(context.Background(), ingress)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(ingress)
	return ingress
}
func (s *ControllerManagerTestSuiteBase) GetAPIServerService() *corev1.Service {
	service := &corev1.Service{}
	s.Require().NoError(wait.PollUntilContextTimeout(context.Background(),
		waitForCreationInterval,
		waitForCreationTimeout,
		true,
		func(ctx context.Context) (done bool, err error) {
			err = s.Mgr.GetClient().Get(ctx, types.NamespacedName{Name: "kubernetes", Namespace: "default"}, service)
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return true, nil
		}),
	)
	return service
}

func (s *ControllerManagerTestSuiteBase) AddReplicaSet(name string, podIps []string, podLabels map[string]string) (*appsv1.ReplicaSet, []*corev1.Pod) {
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("replicaset-%s", name), Namespace: s.TestNamespace},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: lo.ToPtr(int32(len(podIps))),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.TestNamespace, Labels: podLabels},
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
	err := s.Mgr.GetClient().Create(context.Background(), replicaSet)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(replicaSet)

	pods := make([]*corev1.Pod, 0)
	for i, ip := range podIps {
		pod := s.AddPod(fmt.Sprintf("%s-%d", name, i), ip, podLabels, []metav1.OwnerReference{
			{
				APIVersion:         "apps/v1",
				Kind:               "ReplicaSet",
				BlockOwnerDeletion: lo.ToPtr(true),
				Controller:         lo.ToPtr(true),
				Name:               replicaSet.Name,
				UID:                replicaSet.UID,
			},
		})
		pods = append(pods, pod)
	}

	return replicaSet, pods
}

func (s *ControllerManagerTestSuiteBase) AddDeployment(name string, podIps []string, podLabels map[string]string) (*appsv1.Deployment, []*corev1.Pod) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("deployment-%s", name), Namespace: s.TestNamespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: lo.ToPtr(int32(len(podIps))),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.TestNamespace, Labels: podLabels},
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
	err := s.Mgr.GetClient().Create(context.Background(), deployment)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(deployment)

	replicaSet, pods := s.AddReplicaSet(name, podIps, podLabels)
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
	err = s.Mgr.GetClient().Update(context.Background(), replicaSet)
	s.Require().NoError(err)

	return deployment, pods
}

func (s *ControllerManagerTestSuiteBase) AddDeploymentWithService(name string, podIps []string, podLabels map[string]string, serviceIp string) (*appsv1.Deployment, *corev1.Service, []*corev1.Pod) {
	deployment, pods := s.AddDeployment(name, podIps, podLabels)
	service := s.AddService(name, podLabels, serviceIp, pods)
	return deployment, service, pods
}

func (s *ControllerManagerTestSuiteBase) AddDeploymentWithIngressService(name string, podIps []string, podLabels map[string]string, serviceIp string, externalIP string, port int) (*appsv1.Deployment, *corev1.Service, []*corev1.Pod) {
	deployment, pods := s.AddDeployment(name, podIps, podLabels)
	service := s.AddServiceWithIngress(name, podLabels, serviceIp, externalIP, pods, port)
	return deployment, service, pods
}

func (s *ControllerManagerTestSuiteBase) AddDaemonSet(name string, podIps []string, podLabels map[string]string) (*appsv1.DaemonSet, []*corev1.Pod) {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("daemonset-%s", name), Namespace: s.TestNamespace},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.TestNamespace, Labels: podLabels},
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
	err := s.Mgr.GetClient().Create(context.Background(), daemonSet)
	s.Require().NoError(err)

	s.waitForObjectToBeCreated(daemonSet)

	pods := make([]*corev1.Pod, 0)
	for i, ip := range podIps {
		pod := s.AddPod(fmt.Sprintf("%s-%d", name, i), ip, podLabels, []metav1.OwnerReference{
			{
				APIVersion:         "apps/v1",
				Kind:               "DaemonSet",
				BlockOwnerDeletion: lo.ToPtr(true),
				Controller:         lo.ToPtr(true),
				Name:               daemonSet.Name,
				UID:                daemonSet.UID,
			},
		})
		pods = append(pods, pod)
	}

	return daemonSet, pods
}

func (s *ControllerManagerTestSuiteBase) AddDaemonSetWithService(name string, podIps []string, podLabels map[string]string, serviceIp string) (*appsv1.DaemonSet, *corev1.Service) {
	daemonSet, pods := s.AddDaemonSet(name, podIps, podLabels)
	service := s.AddService(name, podLabels, serviceIp, pods)
	return daemonSet, service
}
