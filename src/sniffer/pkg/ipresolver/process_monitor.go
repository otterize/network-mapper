package ipresolver

import (
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

type ProcessMonitor struct {
	processes      sets.Set[int64]
	onProcNew      utils.ProcessScanCallback
	onProcExit     utils.ProcessScanCallback
	forEachProcess utils.ProcessScanner
}

func NewProcessMonitor(
	onProcNew utils.ProcessScanCallback,
	onProcExit utils.ProcessScanCallback,
	forEachProcess utils.ProcessScanner,
) *ProcessMonitor {
	return &ProcessMonitor{
		processes:      sets.New[int64](),
		onProcNew:      onProcNew,
		onProcExit:     onProcExit,
		forEachProcess: forEachProcess,
	}
}

func (pm *ProcessMonitor) Poll() error {
	processesSeenLastTime := pm.processes.Clone()
	pm.processes = sets.New[int64]()

	err := pm.forEachProcess(func(pid int64, pDir string) {
		if !processesSeenLastTime.Has(pid) {
			pm.onProcNew(pid, pDir)
		}
		pm.processes.Insert(pid)
	})
	if err != nil {
		return err
	}

	exitedProcesses := processesSeenLastTime.Difference(pm.processes)
	for _, pid := range exitedProcesses.UnsortedList() {
		pm.onProcExit(pid, "")
	}

	return nil
}
