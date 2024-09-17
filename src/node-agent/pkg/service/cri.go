package service

import (
	"fmt"
	cri "github.com/otterize/network-mapper/src/shared/criclient"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	internalapi "k8s.io/cri-api/pkg/apis"
	"k8s.io/klog/v2"
	"os"
	"time"
)

const CRISocketsPath = "/run/cri"

// CreateCRIClientOrDie tries to connect to CRI sockets in CRISocketsPath
// and returns a RuntimeService for the first valid socket found. If
// none found, the function panics.
// This is done to handle multiple possible locations of CRI sockets, as
// we cannot know during deployment which container runtime will be used.
func CreateCRIClientOrDie() internalapi.RuntimeService {
	logger := klog.Background()

	endpoints, err := getPossibleCRISockets(CRISocketsPath)

	if err != nil {
		logrus.WithError(err).Panic("failed to list CRI sockets")
	}

	for _, endpoint := range endpoints {
		logrus.Debugf("Trying CRI socket: %s", endpoint)

		criClient, err := cri.NewRemoteRuntimeService(
			fmt.Sprintf("unix://%s", endpoint),
			time.Second*5,
			&logger,
		)

		if err == nil {
			logrus.Infof("Connected to CRI socket: %s", endpoint)
			return criClient
		}
	}

	logrus.Fatal("Failed to connect to any CRI socket")
	return nil
}

func getPossibleCRISockets(basePath string) ([]string, error) {
	sockets, err := os.ReadDir(basePath)

	if err != nil {
		return nil, err
	}

	return lo.Map(sockets, func(info os.DirEntry, _ int) string {
		return basePath + "/" + info.Name()
	}), nil
}
