package reconcilers

import (
	"context"
	bpfmanclient "github.com/bpfman/bpfman/clients/gobpfman/v1"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/node-agent/pkg/container"
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
	bpfmanClient      bpfmanclient.BpfmanClient
	containersManager *container.ContainerManager
}

func NewEBPFReconciler(
	client client.Client,
	bpfmanClient bpfmanclient.BpfmanClient,
	containerManager *container.ContainerManager,
) *EBPFReconciler {
	return &EBPFReconciler{
		client:            client,
		bpfmanClient:      bpfmanClient,
		containersManager: containerManager,
	}
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
		containerInfo, err := r.containersManager.GetContainerInfo(ctx, container.ContainerID)

		if err != nil {
			return reconcile.Result{}, errors.Wrap(err)
		}
		_ = r.loadBpfProgramToContainer(ctx, containerInfo)
	}

	return reconcile.Result{}, nil
}

func (r *EBPFReconciler) loadBpfProgramToContainer(ctx context.Context, containerInfo container.ContainerInfo) error {
	fnName := "SSL_write"
	pid := containerInfo.GetPID()

	_, err := r.bpfmanClient.Load(
		ctx,
		&bpfmanclient.LoadRequest{
			Name:        "uprobe_counter",
			ProgramType: Kprobe,
			Attach: &bpfmanclient.AttachInfo{
				Info: &bpfmanclient.AttachInfo_UprobeAttachInfo{
					UprobeAttachInfo: &bpfmanclient.UprobeAttachInfo{
						FnName:       &fnName,
						Target:       "libssl",
						ContainerPid: &pid,
					},
				},
			},
			Bytecode: &bpfmanclient.BytecodeLocation{
				Location: &bpfmanclient.BytecodeLocation_File{
					File: "/otterize/ebpf/uprobe-counter/bpf_x86_bpfel.o",
				},
			},
		},
	)

	if err != nil {
		return errors.Wrap(err)
	}

	logrus.WithField("containerId", containerInfo.GetID()).Info("Loaded program")

	return nil
}
