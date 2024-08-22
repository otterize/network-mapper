package reconcilers

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/ebpf/openssl"
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

const (
	Kprobe     uint32 = 2
	Tc         uint32 = 3
	Tracepoint uint32 = 5
	Xdp        uint32 = 6
	Tracing    uint32 = 26
)

type EBPFReconciler struct {
	client            client.Client
	containersManager *container.ContainerManager
	tracer            ebpf.Tracer
	stop              bool
}

func NewEBPFReconciler(
	client client.Client,
	containerManager *container.ContainerManager,
) *EBPFReconciler {
	return &EBPFReconciler{
		client:            client,
		containersManager: containerManager,
		tracer:            ebpf.NewTracer(),
	}
}

func (r *EBPFReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&corev1.Pod{}, &handler.EnqueueRequestForObject{}).
		Named("ebpf-reconciler").
		Complete(r)
}

func (r *EBPFReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if r.stop {
		return reconcile.Result{}, nil
	}

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

	for _, container := range pod.Status.ContainerStatuses[:1] {
		containerInfo, err := r.containersManager.GetContainerInfo(ctx, container.ContainerID)

		if err != nil {
			return reconcile.Result{}, errors.Wrap(err)
		}
		err = r.loadBpfProgramToContainer(ctx, containerInfo)

		if err != nil {
			return reconcile.Result{}, errors.Wrap(err)
		}

		r.stop = true
	}

	return reconcile.Result{}, nil
}

func (r *EBPFReconciler) loadBpfProgramToContainer(ctx context.Context, containerInfo container.ContainerInfo) error {
	logrus.WithContext(ctx).
		WithField("pid", containerInfo.GetPID()).
		Warningf("Loading %s", openssl.BpfSpecs.OtterizeSSL_write.Name)

	err := r.tracer.AttachToOpenSSL(containerInfo.GetPID())

	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}
