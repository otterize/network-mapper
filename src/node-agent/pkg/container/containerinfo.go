package container

type ContainerInfo struct {
	Id    string
	Pid   int    `json:"pid"`
	PodId string `json:"sandboxId"`
	PodIP string
}
