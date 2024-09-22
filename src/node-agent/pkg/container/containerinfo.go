package container

import (
	"github.com/otterize/network-mapper/src/bintools/bininfo"
	"k8s.io/apimachinery/pkg/types"
)

type ContainerInfo struct {
	Id             string
	Pid            int    `json:"pid"`
	PodId          string `json:"sandboxId"`
	PodIP          string
	PodName        types.NamespacedName
	ExecutableInfo ExecutableInfo
}

type ExecutableInfo struct {
	Arch     bininfo.Arch
	Language bininfo.SourceLanguage
}
