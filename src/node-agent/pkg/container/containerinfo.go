package container

type ContainerInfo interface {
	GetID() string
	GetPID() int32
}

type criContainerInfo struct {
	Id  string
	Pid int32 `json:"pid"`
}

func (c criContainerInfo) GetID() string {
	return c.Id
}

func (c criContainerInfo) GetPID() int32 {
	return c.Pid
}
