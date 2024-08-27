package container

import "github.com/otterize/network-mapper/src/bintools/bininfo"

type ContainerInfo interface {
	GetID() string
	GetPID() int
	GetExecInfo() ExecutableInfo
}

type criContainerInfo struct {
	Id  string
	Pid int `json:"pid"`
	ExecutableInfo ExecutableInfo
}

type ExecutableInfo struct {
	Arch     bininfo.Arch
	Language bininfo.SourceLanguage
}

func (c criContainerInfo) GetID() string {
	return c.Id
}

func (c criContainerInfo) GetPID() int {
	return c.Pid
}

func (c criContainerInfo) GetExecInfo() ExecutableInfo {
	return c.ExecutableInfo
}
