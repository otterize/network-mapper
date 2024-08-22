package container

type ContainerInfo interface {
	GetID() string
	GetPID() int
}

type criContainerInfo struct {
	Id  string
	Pid int `json:"pid"`
}

func (c criContainerInfo) GetID() string {
	return c.Id
}

func (c criContainerInfo) GetPID() int {
	return c.Pid
}
