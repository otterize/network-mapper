package ipresolver

import (
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"
	"github.com/sirupsen/logrus"
)

type ProcFSIPResolverEntry struct {
	IPAddr          string
	Hostname        string
	ProcessRefCount int
}

type ProcFSIPResolver struct {
	byAddr  map[string]*ProcFSIPResolverEntry
	byPid   map[int64]*ProcFSIPResolverEntry
	monitor *ProcessMonitor
}

func NewProcFSIPResolver() *ProcFSIPResolver {
	r := ProcFSIPResolver{
		monitor: nil,
		byAddr:  make(map[string]*ProcFSIPResolverEntry),
		byPid:   make(map[int64]*ProcFSIPResolverEntry),
	}
	r.monitor = NewProcessMonitor(r.onProcessNew, r.onProcessExit, utils.ScanProcDirProcesses)

	return &r
}

func (r *ProcFSIPResolver) ResolveIP(ipaddr string) (hostname string, ok bool) {
	if hostInfo, ok := r.byAddr[ipaddr]; ok {
		return hostInfo.Hostname, true
	}
	return "", false
}

func (r *ProcFSIPResolver) Refresh() error {
	return r.monitor.Poll()
}

func (r *ProcFSIPResolver) onProcessNew(pid int64, pDir string) (err error) {
	var hostname, ipaddr string
	hostname, err = utils.ExtractProcessHostname(pDir)
	if err != nil {
		logrus.Debugf("Failed to extract hostname for process %d: %v", pid, err)
		return errors.Wrap(err)
	}

	ipaddr, err = utils.ExtractProcessIPAddr(pDir)
	if err != nil {
		logrus.Debugf("Failed to extract IP address for process %d: %v", pid, err)
		return errors.Wrap(err)
	}

	if entry, ok := r.byAddr[ipaddr]; ok {
		if entry.Hostname == hostname {
			// Already mapped to this hostname, add another process reference
			r.byPid[pid] = entry
			entry.ProcessRefCount++
			logrus.Debugf("Mapping %s:%s already exists, increased refcount to %d", ipaddr, hostname, entry.ProcessRefCount)
			return nil
		} else {
			// Shouldn't happen - it could happen if an ip replaces its pod very fast and the current single scan sees the new process and not the older one
			logrus.Warnf("IP mapping conflict: %s got new hostname %s, but already mapped to %s. Would use the newer hostname", ipaddr, hostname, entry.Hostname)
			// For now, treat it as a new IP mapping (make sure at exit to decrement ref count only if hostname matches)
		}
	}

	logrus.Debugf("Found new mapping %s:%s", ipaddr, hostname)
	newEntry := &ProcFSIPResolverEntry{
		IPAddr:          ipaddr,
		Hostname:        hostname,
		ProcessRefCount: 1,
	}
	r.byPid[pid] = newEntry
	r.byAddr[ipaddr] = newEntry
	return nil
}

func (r *ProcFSIPResolver) onProcessExit(pid int64, _ string) error {
	if entry, ok := r.byPid[pid]; !ok {
		// Shouldn't happen
		logrus.Debugf("Unknown process %d exited", pid)
		return nil
	} else {
		entry.ProcessRefCount--
		if entry.ProcessRefCount == 0 {
			// Should remove mapping, but validate this process actually holds the newest mapping
			if r.byAddr[entry.IPAddr] == entry {
				logrus.Debugf("Removing IP mapping %s:%s", entry.IPAddr, entry.Hostname)
				delete(r.byAddr, entry.IPAddr)
			}
		}

		// Remove process from pid map
		delete(r.byPid, pid)
	}
	return nil
}
