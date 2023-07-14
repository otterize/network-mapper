package ipresolver

import (
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
	"time"
)

// ProcessMonitorCallback Should be idempotent on failures because retried on error
type ProcessMonitorCallback func(pid int64, pDir string) error

type ProcessMonitor struct {
	processes      sets.Set[int64]
	onProcNew      ProcessMonitorCallback
	onProcExit     ProcessMonitorCallback
	forEachProcess utils.ProcessScanner
}

func NewProcessMonitor(
	onProcNew ProcessMonitorCallback,
	onProcExit ProcessMonitorCallback,
	forEachProcess utils.ProcessScanner,
) *ProcessMonitor {
	return &ProcessMonitor{
		processes:      sets.New[int64](),
		onProcNew:      onProcNew,
		onProcExit:     onProcExit,
		forEachProcess: forEachProcess,
	}
}

func (pm *ProcessMonitor) retryCallbacks(callbacks []func() error) {
	// Retry handling failed processes (to mitigate failing to handle partly initiated /proc/$pid dirs)
	MaxRetries := 3
	for i := 0; i < MaxRetries && len(callbacks) > 0; i++ {
		failed := make([]func() error, 0)
		for _, callback := range callbacks {
			if err := callback(); err != nil {
				failed = append(failed, callback)
			}
		}
		callbacks = failed
		if i < MaxRetries-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (pm *ProcessMonitor) Poll() error {
	processesSeenLastTime := pm.processes.Clone()
	pm.processes = sets.New[int64]()
	procNewCallbacks := make([]func() error, 0)

	if err := pm.forEachProcess(func(pid int64, pDir string) {
		if !processesSeenLastTime.Has(pid) {
			procNewCallbacks = append(procNewCallbacks, func() error {
				return pm.onProcNew(pid, pDir)
			})
		}
		pm.processes.Insert(pid)
	}); err != nil {
		return err
	}

	pm.retryCallbacks(procNewCallbacks)

	exitedProcesses := processesSeenLastTime.Difference(pm.processes)
	for _, pid := range exitedProcesses.UnsortedList() {
		_ = pm.onProcExit(pid, "")
	}

	return nil
}
