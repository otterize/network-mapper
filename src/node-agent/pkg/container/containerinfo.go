package container

import "github.com/otterize/network-mapper/src/bintools/bininfo"

type ContainerInfo struct {
	Id             string
	Pid            int    `json:"pid"`
	PodId          string `json:"sandboxId"`
	PodIP          string
	ExecutableInfo ExecutableInfo
}

type ExecutableInfo struct {
	Arch     bininfo.Arch
	Language bininfo.SourceLanguage
}
