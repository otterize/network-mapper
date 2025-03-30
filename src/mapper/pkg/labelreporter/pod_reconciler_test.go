package labelreporter

import (
	"context"
	serviceidresolvermocks "github.com/otterize/intents-operator/src/shared/serviceidresolver/mocks"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"

	cloudclientmocks "github.com/otterize/network-mapper/src/mapper/pkg/cloudclient/mocks"
	"github.com/otterize/network-mapper/src/mapper/pkg/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodReconcilerTestSuite struct {
	suite.Suite
	cloudClient       *cloudclientmocks.MockCloudClient
	k8sClient         *mocks.K8sClient
	reconciler        *PodReconciler
	serviceIDResolver *serviceidresolvermocks.MockServiceResolver
}

func (s *PodReconcilerTestSuite) SetupTest() {
	controller := gomock.NewController(s.T())
	s.cloudClient = cloudclientmocks.NewMockCloudClient(controller)
	s.k8sClient = mocks.NewK8sClient(controller)
	s.serviceIDResolver = serviceidresolvermocks.NewMockServiceResolver(controller)
	s.reconciler, _ = NewPodReconciler(s.k8sClient, s.cloudClient, s.serviceIDResolver)
}

func (s *PodReconcilerTestSuite) disableSyncOnce() {
	s.reconciler.once.Do(func() {})
}

func (s *PodReconcilerTestSuite) TestPodReconciler_Reconcile() {
	s.disableSyncOnce()
	testPodName := "test-pod"
	testNamespace := "test-namespace"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
			Labels:    map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: testNamespace, Name: testPodName},
	}

	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, pod *corev1.Pod, _ ...any) error {
			testPod.DeepCopyInto(pod)
			return nil
		})

	s.serviceIDResolver.EXPECT().ResolvePodToServiceIdentity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, pod *corev1.Pod) (serviceidentity.ServiceIdentity, error) {
			return serviceidentity.ServiceIdentity{
				Name:      testPodName,
				Namespace: testNamespace,
			}, nil
		})

	expectedWorkloadLabelsInput := []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      testPodName,
				Namespace: testNamespace,
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key1", Value: nilable.From("value1")},
					{Key: "key2", Value: nilable.From("value2")},
				},
			},
		},
	}

	s.cloudClient.EXPECT().ReportWorkloadsLabels(gomock.Any(), expectedWorkloadLabelsInput).Return(nil)

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Require().True(res.IsZero())
}

func (s *PodReconcilerTestSuite) TestPodReconciler_Cache() {
	s.disableSyncOnce()
	testPodName := "test-pod"
	testNamespace := "test-namespace"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
			Labels:    map[string]string{"key1": "value1", "key2": "value2"},
		},
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: testNamespace, Name: testPodName},
	}

	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, pod *corev1.Pod, _ ...any) error {
			testPod.DeepCopyInto(pod)
			return nil
		})

	s.serviceIDResolver.EXPECT().ResolvePodToServiceIdentity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, pod *corev1.Pod) (serviceidentity.ServiceIdentity, error) {
			return serviceidentity.ServiceIdentity{
				Name:      testPodName,
				Namespace: testNamespace,
			}, nil
		})

	expectedWorkloadLabelsInput := []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      testPodName,
				Namespace: testNamespace,
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key1", Value: nilable.From("value1")},
					{Key: "key2", Value: nilable.From("value2")},
				},
			},
		},
	}

	s.cloudClient.EXPECT().ReportWorkloadsLabels(gomock.Any(), expectedWorkloadLabelsInput).Return(nil)

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Require().True(res.IsZero())

	// Second reconcile should not trigger ReportWorkloadsLabels again

	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, pod *corev1.Pod, _ ...any) error {
			testPod.DeepCopyInto(pod)
			return nil
		})

	s.serviceIDResolver.EXPECT().ResolvePodToServiceIdentity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, pod *corev1.Pod) (serviceidentity.ServiceIdentity, error) {
			return serviceidentity.ServiceIdentity{
				Name:      testPodName,
				Namespace: testNamespace,
			}, nil
		})
	res, err = s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Require().True(res.IsZero())

	// Third reconcile with different labels should trigger ReportWorkloadsLabels again

	testPod.Labels = map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}
	expectedWorkloadLabelsInput = []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      testPodName,
				Namespace: testNamespace,
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key1", Value: nilable.From("value1")},
					{Key: "key2", Value: nilable.From("value2")},
					{Key: "key3", Value: nilable.From("value3")},
				},
			},
		},
	}

	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, pod *corev1.Pod, _ ...any) error {
			testPod.DeepCopyInto(pod)
			return nil
		})
	s.serviceIDResolver.EXPECT().ResolvePodToServiceIdentity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, pod *corev1.Pod) (serviceidentity.ServiceIdentity, error) {
			return serviceidentity.ServiceIdentity{
				Name:      testPodName,
				Namespace: testNamespace,
			}, nil
		})

	s.cloudClient.EXPECT().ReportWorkloadsLabels(gomock.Any(), expectedWorkloadLabelsInput).Return(nil)

	res, err = s.reconciler.Reconcile(context.Background(), req)

}

func (s *PodReconcilerTestSuite) TestPodReconciler_SyncOnce() {
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
			{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "namespace1", Labels: map[string]string{"key1": "value1"}}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "namespace1", Labels: map[string]string{"key2": "value2"}}},
		},
	}
	podsNamespace2 := &corev1.PodList{
		Items: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "pod3", Namespace: "namespace2", Labels: map[string]string{"key3": "value3"}}},
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
			return serviceidentity.ServiceIdentity{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			}, nil
		}).Times(3)

	// Mock cloud client report
	expectedWorkloadLabelsInput1 := []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      "pod1",
				Namespace: "namespace1",
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key1", Value: nilable.From("value1")},
				},
			},
		},
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      "pod2",
				Namespace: "namespace1",
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key2", Value: nilable.From("value2")},
				},
			},
		},
	}
	s.cloudClient.EXPECT().ReportWorkloadsLabels(gomock.Any(), expectedWorkloadLabelsInput1).Return(nil)

	expectedWorkloadLabelsInput2 := []cloudclient.ReportServiceMetadataInput{
		{
			Identity: cloudclient.ServiceIdentityInput{
				Name:      "pod3",
				Namespace: "namespace2",
			},
			Metadata: cloudclient.ServiceMetadataInput{
				Labels: []cloudclient.LabelInput{
					{Key: "key3", Value: nilable.From("value3")},
				},
			},
		},
	}
	s.cloudClient.EXPECT().ReportWorkloadsLabels(gomock.Any(), expectedWorkloadLabelsInput2).Return(nil)

	// Call reconcile with deleted pod so it will only do the sync once
	req := ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: "test-namespace", Name: "test-pod"},
	}
	deletedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-namespace",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}
	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, pod *corev1.Pod, _ ...any) error {
			deletedPod.DeepCopyInto(pod)
			return nil
		})

	res, err := s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Require().True(res.IsZero())

	// Call reconcile with a pod1 from namespace1 see it will not trigger the sync once but also not report labels because of the cache
	req = ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: "namespace1", Name: "pod1"},
	}
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "namespace1",
			Labels:    map[string]string{"key1": "value1"},
		},
	}
	s.k8sClient.EXPECT().Get(gomock.Any(), req.NamespacedName, gomock.Any()).DoAndReturn(
		func(ctx context.Context, name types.NamespacedName, pod *corev1.Pod, _ ...any) error {
			pod1.DeepCopyInto(pod)
			return nil
		})

	s.serviceIDResolver.EXPECT().ResolvePodToServiceIdentity(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, pod *corev1.Pod) (serviceidentity.ServiceIdentity, error) {
			return serviceidentity.ServiceIdentity{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			}, nil
		})

	res, err = s.reconciler.Reconcile(context.Background(), req)
	s.NoError(err)
	s.Require().True(res.IsZero())
}

func TestPodReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(PodReconcilerTestSuite))
}
