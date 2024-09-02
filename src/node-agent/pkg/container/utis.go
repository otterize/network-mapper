package container

import "fmt"

func GetContainerExecPath(pid int) string {
	return fmt.Sprintf("/host/proc/%d/exe", pid)
}
