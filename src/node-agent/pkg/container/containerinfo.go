package container

type ContainerInfo interface {
	GetID() string
	GetPID() uint32
}

type criContainerInfo struct {
	Id  string
	Pid uint32 `json:"pid"`
}

func (c criContainerInfo) GetID() string {
	return c.Id
}

func (c criContainerInfo) GetPID() uint32 {
	return c.Pid
}
