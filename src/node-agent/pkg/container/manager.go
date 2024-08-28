package container

import (
	"context"
	"encoding/json"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	internalapi "k8s.io/cri-api/pkg/apis"
	"strings"
)

type ContainerManager struct {
	criClient internalapi.RuntimeService
}

func NewContainerManager(criClient internalapi.RuntimeService) *ContainerManager {
	return &ContainerManager{
		criClient: criClient,
	}
}

func (m *ContainerManager) GetContainerInfo(ctx context.Context, pod v1.Pod, containerID string) (ContainerInfo, error) {
	containerType, containerId, found := strings.Cut(containerID, "://")

	if !found {
		return ContainerInfo{}, errors.Errorf("Failed to parse container ID: %s", containerID)
	}

	logrus.WithField("containerType", containerType).
		WithField("containerId", containerId).
		Debug("Getting container info")

	resp, err := m.criClient.ContainerStatus(ctx, containerId, true)

	if err != nil {
		return ContainerInfo{}, errors.Wrap(err)
	}

	if resp.Info == nil {
		return ContainerInfo{}, errors.Errorf("invalid container info for %s", containerId)
	}

	if _, ok := resp.Info["info"]; !ok {
		return ContainerInfo{}, errors.Errorf("invalid container info for %s", containerId)
	}

	var info ContainerInfo
	err = json.Unmarshal([]byte(resp.Info["info"]), &info)

	if err != nil {
		return ContainerInfo{}, errors.Wrap(err)
	}

	info.Id = resp.Status.Id
	info.PodIP = pod.Status.PodIP

	logrus.WithField("containerId", info.Id).WithField("info", info).Debug("Got container info")

	return info, nil
}
