package reconcilers

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
	"github.com/otterize/network-mapper/src/node-agent/pkg/ebpf"
	"github.com/otterize/network-mapper/src/node-agent/pkg/labels"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type EBPFReconciler struct {
	client            client.Client
	containersManager *container.ContainerManager
	tracer            *ebpf.Tracer
	eventReader       *ebpf.EventReader
}

func NewEBPFReconciler(
	client client.Client,
	containerManager *container.ContainerManager,
	finder *kubefinder.KubeFinder,
) (*EBPFReconciler, error) {
	eventReader, err := ebpf.NewEventReader(openssl.BpfObjects.SslEvents)

	if err != nil {
		return nil, errors.Wrap(err)
	}

	eventReader.Start()

	return &EBPFReconciler{
		client:            client,
		containersManager: containerManager,
		tracer:            ebpf.NewTracer(eventReader),
		eventReader:       eventReader,
	}, nil
}

func (r *EBPFReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&corev1.Pod{}, &handler.EnqueueRequestForObject{}).
		Named("ebpf-reconciler").
		Complete(r)
}

func (r *EBPFReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := logrus.WithContext(ctx).
		WithField("namespace", req.Namespace).
		WithField("podName", req.Name)

	pod := corev1.Pod{}

	if err := r.client.Get(ctx, req.NamespacedName, &pod); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, errors.Wrap(err)
	}

	if pod.Status.Phase != corev1.PodRunning {
		logger.Debug("Pod is not running, skipping")
		return reconcile.Result{}, nil
	}

	if !kubeutils.IsEnabledByLabel(pod.Labels, labels.EBPFVisibilityLabelKey) {
		return reconcile.Result{}, nil
	}

	for _, container := range pod.Status.ContainerStatuses {
		containerInfo, err := r.containersManager.GetContainerInfo(ctx, pod, container.ContainerID)

		if err != nil {
			return reconcile.Result{}, errors.Wrap(err)
		}

		err = r.loadBpfProgramToContainer(ctx, containerInfo)

		if err != nil {
			return reconcile.Result{}, errors.Wrap(err)
		}
	}

	return reconcile.Result{}, nil
}

func (r *EBPFReconciler) loadBpfProgramToContainer(ctx context.Context, containerInfo container.ContainerInfo) error {
	err := r.tracer.AttachToOpenSSL(containerInfo)

	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}
