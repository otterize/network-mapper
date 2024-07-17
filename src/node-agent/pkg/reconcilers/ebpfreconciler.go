package reconcilers

import (
	"context"
	"encoding/json"
	bpfmanclient "github.com/bpfman/bpfman/clients/gobpfman/v1"
	"github.com/otterize/intents-operator/src/shared/errors"
	cri "github.com/otterize/network-mapper/src/shared/criclient"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

const (
	Kprobe     uint32 = 2
	Tc         uint32 = 3
	Tracepoint uint32 = 5
	Xdp        uint32 = 6
	Tracing    uint32 = 26
)

type EBPFReconciler struct {
	client       client.Client
	bpfmanClient bpfmanclient.BpfmanClient
}

func NewEBPFReconciler(client client.Client, bpfmanClient *bpfmanclient.BpfmanClient) *EBPFReconciler {
	return &EBPFReconciler{
		client:       client,
		bpfmanClient: *bpfmanClient,
	}
}

func (r *EBPFReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logrus.WithContext(ctx).
		WithField("namespace", req.Namespace).
		WithField("name", req.Name).
		Info("Reconciling EBPF")

	pod := corev1.Pod{}

	if err := r.client.Get(ctx, req.NamespacedName, &pod); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, errors.Wrap(err)
	}

	if pod.Status.Phase != corev1.PodRunning {
		return reconcile.Result{}, nil
	}

	_, hasEBPFLabel := pod.Labels["ebpf"]

	if !hasEBPFLabel {
		return reconcile.Result{}, nil
	}

	for _, container := range pod.Status.ContainerStatuses {
		_ = r.loadBpfProgramToContainer(ctx, &container)
	}

	return reconcile.Result{}, nil
}

func (r *EBPFReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&corev1.Pod{}, &handler.EnqueueRequestForObject{}).
		Named("ebpf-reconciler").
		Complete(r)
}

func (r *EBPFReconciler) loadBpfProgramToContainer(ctx context.Context, container *corev1.ContainerStatus) error {
	logger := klog.Background()

	// TODO: move to main
	criClient, err := cri.NewRemoteRuntimeService(
		"unix:///var/run/containerd/containerd.sock",
		time.Second*5,
		&logger,
	)

	if err != nil {
		return errors.Wrap(err)
	}

	// form is containerd://<container-id>
	_, containerId, found := strings.Cut(container.ContainerID, "://")

	if !found {
		return errors.Errorf("Failed to parse container ID: %s", container.ContainerID)
	}

	criResp, err := criClient.ContainerStatus(ctx, containerId, true)

	if err != nil {
		return errors.Wrap(err)
	}

	containerInfo, found := criResp.Info["info"]

	if !found {
		return errors.Errorf("Failed to get container info: %s", containerId)
	}

	type ContainerInfo struct {
		Pid int32 `json:"pid"`
	}

	var containerInfoStruct ContainerInfo
	err = json.Unmarshal([]byte(containerInfo), &containerInfoStruct)

	if err != nil {
		logrus.WithError(err).Error("Failed to unmarshal container info")
		return err
	}

	logrus.WithField("pid", containerInfoStruct.Pid).WithError(err).Info("Container PID")

	//fnName := "SSL_write"
	//resp, err := r.bpfmanClient.Load(
	//	ctx,
	//	&bpfmanclient.LoadRequest{
	//		Name:        "uprobe_counter",
	//		ProgramType: Kprobe,
	//		Attach: &bpfmanclient.AttachInfo{
	//			Info: &bpfmanclient.AttachInfo_UprobeAttachInfo{
	//				UprobeAttachInfo: &bpfmanclient.UprobeAttachInfo{
	//					FnName:       &fnName,
	//					Target:       "libssl",
	//					ContainerPid: &containerInfoStruct.Pid,
	//				},
	//			},
	//		},
	//		Bytecode: &bpfmanclient.BytecodeLocation{
	//			Location: &bpfmanclient.BytecodeLocation_File{
	//				File: "/otterize/ebpf/uprobe-counter/bpf_x86_bpfel.o",
	//			},
	//		},
	//	},
	//)

	err = nil

	if err != nil {
		return errors.Wrap(err)
	}

	logrus.WithField("containerId", containerId).Info("Loaded program")

	return nil
}
