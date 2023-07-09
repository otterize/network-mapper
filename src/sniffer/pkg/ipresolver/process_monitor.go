package ipresolver

import "github.com/otterize/network-mapper/src/sniffer/pkg/utils"

type ProcessMonitor struct {
	processes  map[int64]interface{}
	onProcNew  utils.ProcessScanCallback
	onProcExit utils.ProcessScanCallback
	done       chan bool
}

func NewProcessMonitor(onProcNew, onProcExit utils.ProcessScanCallback) *ProcessMonitor {
	return &ProcessMonitor{
		processes:  make(map[int64]interface{}),
		onProcNew:  onProcNew,
		onProcExit: onProcExit,
		done:       nil,
	}
}

func (pm *ProcessMonitor) Poll() error {
	oldProcesses := make(map[int64]bool)
	for pid := range pm.processes {
		oldProcesses[pid] = false
	}

	err := utils.ScanProcDirProcesses(func(pid int64, pDir string) {
		if _, ok := pm.processes[pid]; !ok {
			// New process
			pm.onProcNew(pid, pDir)
			pm.processes[pid] = nil
		} else {
			// Existing process
			oldProcesses[pid] = true
		}
	})
	if err != nil {
		return err
	}

	for pid := range oldProcesses {
		if !oldProcesses[pid] {
			pm.onProcExit(pid, "")
			// Process no longer exists
			delete(pm.processes, pid)
		}
	}

	return nil
}
