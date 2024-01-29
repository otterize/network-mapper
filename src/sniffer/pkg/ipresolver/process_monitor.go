package ipresolver

import (
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

const MaxRetries = 3

// ProcessMonitorCallback Should be idempotent on failures because retried on error
type ProcessMonitorCallback func(pid int64, pDir string) error

type ProcessMonitor struct {
	processes        sets.Set[int64]
	failingProcesses map[int64]int
	onProcNew        ProcessMonitorCallback
	onProcExit       ProcessMonitorCallback
	forEachProcess   utils.ProcessScanner
}

func NewProcessMonitor(
	onProcNew ProcessMonitorCallback,
	onProcExit ProcessMonitorCallback,
	forEachProcess utils.ProcessScanner,
) *ProcessMonitor {
	return &ProcessMonitor{
		processes:        sets.New[int64](),
		failingProcesses: make(map[int64]int),
		onProcNew:        onProcNew,
		onProcExit:       onProcExit,
		forEachProcess:   forEachProcess,
	}
}

func (pm *ProcessMonitor) Poll() error {
	processesSeenLastTime := pm.processes.Clone()
	pm.processes = sets.New[int64]()

	if err := pm.forEachProcess(func(pid int64, pDir string) {
		if !processesSeenLastTime.Has(pid) {
			if err := pm.onProcNew(pid, pDir); err != nil {
				// Failed to handle
				failures := 0
				if _, ok := pm.failingProcesses[pid]; ok {
					failures = pm.failingProcesses[pid]
				}
				failures++
				if failures <= MaxRetries {
					// Try again next interval
					pm.failingProcesses[pid] = failures
					return // Don't insert pid to handled set
				} else {
					logrus.Debugf("Giving up failing process: %d", pid)
					delete(pm.failingProcesses, pid)
				}
			}
		}
		// Shouldn't handle again
		pm.processes.Insert(pid)
	}); err != nil {
		return errors.Wrap(err)
	}

	exitedProcesses := processesSeenLastTime.Difference(pm.processes)
	for _, pid := range exitedProcesses.UnsortedList() {
		_ = pm.onProcExit(pid, "")
	}

	return nil
}
