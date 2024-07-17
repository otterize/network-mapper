package container

import (
	"context"
	"encoding/json"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/sirupsen/logrus"
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

func (m *ContainerManager) GetContainerInfo(ctx context.Context, containerID string) (ContainerInfo, error) {
	containerType, containerId, found := strings.Cut(containerID, "://")

	if !found {
		return nil, errors.Errorf("Failed to parse container ID: %s", containerID)
	}

	logrus.WithField("containerType", containerType).
		WithField("containerId", containerId).
		Debug("Getting container info")

	resp, err := m.criClient.ContainerStatus(ctx, containerId, true)

	if err != nil {
		return nil, errors.Wrap(err)
	}

	if resp.Info == nil {
		return nil, errors.Errorf("invalid container info for %s", containerId)
	}

	if _, ok := resp.Info["info"]; !ok {
		return nil, errors.Errorf("invalid container info for %s", containerId)
	}

	var info criContainerInfo
	err = json.Unmarshal([]byte(resp.Info["info"]), &info)

	if err != nil {
		return nil, errors.Wrap(err)
	}

	info.Id = resp.Status.Id

	return info, nil
}
