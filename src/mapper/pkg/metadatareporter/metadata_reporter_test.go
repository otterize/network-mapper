package metadatareporter

import (
	"context"
	serviceidresolvermocks "github.com/otterize/intents-operator/src/shared/serviceidresolver/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/otterize/nilable"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
)

type MetadataReporterTestSuite struct {
	suite.Suite
	cloudClient       *cloudclientmocks.MockCloudClient
	serviceIDResolver *serviceidresolvermocks.MockServiceResolver
	k8sClient         *mocks.K8sClient
	reporter          *MetadataReporter
}

func (s *MetadataReporterTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.serviceIDResolver = serviceidresolvermocks.NewMockServiceResolver(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.reporter = NewMetadataReporter(s.k8sClient, s.cloudClient, s.serviceIDResolver)
}

func (s *MetadataReporterTestSuite) TestReportMetadata() {
	s.reporter.once.Do(func() {})
	// Mock service identity
	serviceIdentity := serviceidentity.ServiceIdentity{
		Name:      "test-service",
		Namespace: "test-namespace",
	}

	// Mock pods resolved from service identity
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "test-namespace",
				Labels:    map[string]string{"key1": "value1"},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "test-namespace",
				Labels:    map[string]string{"key1": "value1"},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.2",
			},
		},
	}

	// expected list endpoints (by pod name) for the service ips
	endpoints := []corev1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: "test-namespace",
			},
		},
	}

	// Mock endpoints for the pods
	for _, pod := range pods {
		// expect client list with pod name (using endpointsPodNamesIndexField)
		// expect with the pod name
		s.k8sClient.EXPECT().List(
			gomock.Any(),
			gomock.Eq(&corev1.EndpointsList{}),
			gomock.Eq(client.InNamespace(pod.Namespace)),
			gomock.Eq(client.MatchingFields{endpointsPodNamesIndexField: pod.Name}),
		).Do(
			func(ctx context.Context, list *corev1.EndpointsList, _ ...any) {
				list.Items = endpoints
			})
	}

	// Mock listing service with the same name as the endpoints
	serviceIPs := []string{"192.168.1.1"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-namespace",
		},
		Spec: corev1.ServiceSpec{
			Type:       corev1.ServiceTypeClusterIP,
			ClusterIPs: serviceIPs,
		},
	}
	s.k8sClient.EXPECT().Get(
		gomock.Any(),
		gomock.Eq(client.ObjectKey{Namespace: serviceIdentity.Namespace, Name: service.Name}),
		gomock.Eq(&corev1.Service{}),
	).Do(
		func(ctx context.Context, key client.ObjectKey, svc *corev1.Service, _ ...any) {
			service.DeepCopyInto(svc)
		})

	// Mock resolver behavior
	s.serviceIDResolver.EXPECT().ResolveServiceIdentityToPodSlice(gomock.Any(), serviceIdentity).Return(pods, true, nil)

	// Mock cloud client behavior
	expectedMetadata := []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      "test-service",
				Namespace: "test-namespace",
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key1", Value: nilable.From("value1")},
				},
				PodIps:     []string{"10.0.0.1", "10.0.0.2"},
				ServiceIps: serviceIPs,
			},
		},
	}
	s.cloudClient.EXPECT().ReportWorkloadsMetadata(gomock.Any(), expectedMetadata).Return(nil)

	// Call ReportMetadata
	err := s.reporter.ReportMetadata(context.Background(), []serviceidentity.ServiceIdentity{serviceIdentity})
	s.NoError(err)
}

func (s *MetadataReporterTestSuite) TestReportMetadata_Once() {
	// Mock namespace list
	namespaces := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "namespace1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "namespace2"}},
		},
	}
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
			namespaces.DeepCopyInto(list.(*corev1.NamespaceList))
			return nil
		})

	// Mock pod list for each namespace
	podsNamespace1 := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: map[string]string{"key1": "value1"}},
				Status: corev1.PodStatus{
					PodIP: "10.0.0.1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace1", Labels: map[string]string{"key2": "value2"}},
				Status: corev1.PodStatus{
					PodIP: "10.0.0.2",
				},
			},
		},
	}

	podsNamespace2 := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "pod3", Namespace: "namespace2", Labels: map[string]string{"key3": "value3"}},
				Status: corev1.PodStatus{
					PodIP: "10.0.0.3",
				},
			},
		},
	}

	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Any(), client.InNamespace("namespace1")).DoAndReturn(
		func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
			podsNamespace1.DeepCopyInto(list.(*corev1.PodList))
			return nil
		})
	s.k8sClient.EXPECT().List(gomock.Any(), gomock.Any(), client.InNamespace("namespace2")).DoAndReturn(
		func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
			podsNamespace2.DeepCopyInto(list.(*corev1.PodList))
			return nil
		})

	// Mock service ID resolver
	s.serviceIDResolver.EXPECT().ResolvePodToServiceIdentity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, pod *corev1.Pod) (serviceidentity.ServiceIdentity, error) {
			si := serviceidentity.ServiceIdentity{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			}
			// Return false to avoid testing all the other logic
			// this will happen in the metadata reporter
			s.serviceIDResolver.EXPECT().ResolveServiceIdentityToPodSlice(gomock.Any(), gomock.Eq(si)).Return(nil, false, nil)
			return si, nil
		}).Times(3)

	s.Require().NoError(s.reporter.ReportMetadata(context.Background(), make([]serviceidentity.ServiceIdentity, 0)))

	// Second call should not do anything
	s.Require().NoError(s.reporter.ReportMetadata(context.Background(), make([]serviceidentity.ServiceIdentity, 0)))
}

func (s *MetadataReporterTestSuite) TestReportMetadata_Cache() {
	s.reporter.once.Do(func() {})

	// Mock service identity
	serviceIdentity := serviceidentity.ServiceIdentity{
		Name:      "test-service",
		Namespace: "test-namespace",
	}

	// Mock pods resolved from service identity
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "test-namespace",
				Labels:    map[string]string{"key1": "value1"},
			},
			Status: corev1.PodStatus{
				PodIP: "10.0.0.1",
			},
		},
	}

	// Mock endpoints for the pods
	endpoints := []corev1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: "test-namespace",
			},
		},
	}

	for _, pod := range pods {
		s.k8sClient.EXPECT().List(
			gomock.Any(),
			gomock.Eq(&corev1.EndpointsList{}),
			gomock.Eq(client.InNamespace(pod.Namespace)),
			gomock.Eq(client.MatchingFields{endpointsPodNamesIndexField: pod.Name}),
		).Do(
			func(ctx context.Context, list *corev1.EndpointsList, _ ...any) {
				list.Items = endpoints
			}).Times(2)
	}

	// Mock listing service with the same name as the endpoints
	serviceIPs := []string{"192.168.1.1"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-namespace",
		},
		Spec: corev1.ServiceSpec{
			Type:       corev1.ServiceTypeClusterIP,
			ClusterIPs: serviceIPs,
		},
	}
	s.k8sClient.EXPECT().Get(
		gomock.Any(),
		gomock.Eq(client.ObjectKey{Namespace: serviceIdentity.Namespace, Name: service.Name}),
		gomock.Eq(&corev1.Service{}),
	).Do(
		func(ctx context.Context, key client.ObjectKey, svc *corev1.Service, _ ...any) {
			service.DeepCopyInto(svc)
		}).Times(2)

	// Mock resolver behavior
	s.serviceIDResolver.EXPECT().ResolveServiceIdentityToPodSlice(gomock.Any(), serviceIdentity).Return(pods, true, nil).Times(2)

	// Mock cloud client behavior for the first call
	expectedMetadata := []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      "test-service",
				Namespace: "test-namespace",
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key1", Value: nilable.From("value1")},
				},
				PodIps:     []string{"10.0.0.1"},
				ServiceIps: serviceIPs,
			},
		},
	}
	// this will happen only once because of the cache
	s.cloudClient.EXPECT().ReportWorkloadsMetadata(gomock.Any(), expectedMetadata).Return(nil).Times(1)

	// First call to ReportMetadata
	err := s.reporter.ReportMetadata(context.Background(), []serviceidentity.ServiceIdentity{serviceIdentity})
	s.NoError(err)

	// Second call to ReportMetadata with the same service identity
	// No cloud client call is expected
	err = s.reporter.ReportMetadata(context.Background(), []serviceidentity.ServiceIdentity{serviceIdentity})
	s.NoError(err)
}

func TestMetadataReporterTestSuite(t *testing.T) {
	suite.Run(t, new(MetadataReporterTestSuite))
}
