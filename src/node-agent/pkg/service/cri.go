package service

import (
	cri "github.com/otterize/network-mapper/src/shared/criclient"
	"github.com/sirupsen/logrus"
	internalapi "k8s.io/cri-api/pkg/apis"
	"k8s.io/klog/v2"
	"time"
)

func CreateCRIClientOrDie() internalapi.RuntimeService {
	logger := klog.Background()

	criClient, err := cri.NewRemoteRuntimeService(
		"unix:///var/run/containerd/containerd.sock",
		time.Second*5,
		&logger,
	)

	if err != nil {
		logrus.WithError(err).Panic("failed to create CRI client")
	}

	return criClient
}
