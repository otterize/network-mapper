package ipresolver

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mpvl/unique"
	"github.com/otterize/network-mapper/src/sniffer/pkg/utils"
	"github.com/sirupsen/logrus"
)

type ProcessMonitor struct {
	processes  map[int64]interface{}
	onProcNew  utils.ProcessScanCallback
	onProcExit utils.ProcessScanCallback
	pollEvent  chan bool // TODO: Replace with sync.Cond if supporting multiple callers is needed
	pollLock   sync.Mutex
	done       chan bool
}

func NewProcessMonitor(onProcNew, onProcExit utils.ProcessScanCallback) *ProcessMonitor {
	return &ProcessMonitor{
		processes:  make(map[int64]interface{}),
		onProcNew:  onProcNew,
		onProcExit: onProcExit,
		pollEvent:  make(chan bool, 1),
		done:       nil,
	}
}

func (pm *ProcessMonitor) Start(intervals ...int) {
	var interval int
	if len(intervals) > 0 {
		interval = intervals[0] * 1000
	} else {
		interval = 500 // default interval
	}

	pm.done = make(chan bool)

	go func() {
		for {
			select {
			case <-pm.done:
				return
			default:
				pm.pollLock.Lock()
				err := pm.poll()

				select { // Raise event unless channel already full
				case pm.pollEvent <- true:
				default:
				}
				pm.pollLock.Unlock()

				if err != nil {
					logrus.Errorf("ProcessMonitor: poll failed: %v", err)
				}
				time.Sleep(time.Duration(interval) * time.Millisecond)
			}
		}
	}()
}

func (pm *ProcessMonitor) Stop() {
	if pm.done != nil {
		pm.done <- true
	}
}

func (pm *ProcessMonitor) WaitForNextPoll() {
	pm.pollLock.Lock() // Lock ensures poll hasn't already started, we're waiting for entirely new refresh
	select {
	case <-pm.pollEvent: // Reset event if it's already set
	default:
	}
	pm.pollLock.Unlock()

	<-pm.pollEvent // Wait for next poll to complete
}

func (pm *ProcessMonitor) poll() error {
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
	r.monitor = NewProcessMonitor(r.onProcessNew, r.onProcessExit)
	r.monitor.Start()
	return &r
}

func (r *ProcFSIPResolver) Stop() {
	r.monitor.Stop()
}

func (r *ProcFSIPResolver) ResolveIP(ipaddr string) (hostname string, err error) {
	if hostInfo, ok := r.byAddr[ipaddr]; ok {
		return hostInfo.Hostname, nil
	}
	return "", errors.New("IP not found")
}

func (r *ProcFSIPResolver) WaitForNextRefresh() {
	r.monitor.WaitForNextPoll()
}

func (r *ProcFSIPResolver) onProcessNew(pid int64, pDir string) {
	hostname, err := extractHostname(pDir)
	if err != nil {
		logrus.Errorf("Failed to extract hostname for process %d: %v", pid, err)
		return
	}

	ipaddr, err := extractIPAddr(pDir)
	if err != nil {
		logrus.Errorf("Failed to extract IP address for process %d: %v", pid, err)
		return
	}

	if entry, ok := r.byAddr[ipaddr]; ok {
		if entry.Hostname == hostname {
			// Already mapped to this hostname, add another process referece
			r.byPid[pid] = entry
			entry.ProcessRefCount++
			return
		} else {
			// Shouldn't happen
			logrus.Warnf("IP mapping conflict: %s got new hostname %s, but already mapped to %s. Would use the newer hostname", ipaddr, hostname, entry.Hostname)
			// For now, treat it as a new IP mapping (make sure at exit to decrement ref count only if hostname matches)
		}
	}

	newEntry := &ProcFSIPResolverEntry{
		IPAddr:          ipaddr,
		Hostname:        hostname,
		ProcessRefCount: 1,
	}
	r.byPid[pid] = newEntry
	r.byAddr[ipaddr] = newEntry
}

func (r *ProcFSIPResolver) onProcessExit(pid int64, _ string) {
	if entry, ok := r.byPid[pid]; !ok {
		// Shouldn't happen
		logrus.Warnf("Unknown process %d exited", pid)
		return
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
}

func extractHostname(pDir string) (string, error) {
	// Read the environment variables from the proc filesystem
	data, err := os.ReadFile(fmt.Sprintf("%s/environ", pDir))
	if err != nil {
		return "", err
	}

	// Split the environment variables by null byte
	envVars := strings.Split(string(data), "\x00")
	for _, envVarLine := range envVars {
		// Split the environment variable line into a name and value
		parts := strings.SplitN(envVarLine, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// If the environment variable name matches the requested one, return its value
		if parts[0] == "HOSTNAME" {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("couldn't find hostname in %s/environ", pDir)

}

func extractIPAddr(pDir string) (string, error) {
	contentBytes, err := os.ReadFile(fmt.Sprintf("%s/net/fib_trie", pDir))
	if err != nil {
		return "", err
	}

	content := string(contentBytes)

	// Regular expression to match the IP addresses labelled as '/32 host LOCAL' but are not loopback addresses
	re := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\s*/32 host LOCAL`)

	matches := re.FindAllStringSubmatch(content, -1)

	ips := make([]string, 0)

	for _, match := range matches {
		if len(match) > 1 && !strings.HasPrefix(match[1], "127.") {
			ips = append(ips, match[1])
		}
	}
	unique.Strings(&ips)

	if len(ips) == 0 {
		return "", errors.New("no IP addresses found")
	}
	if len(ips) > 1 {
		logrus.Warnf("Found multiple IP addresses (%s) in %s", ips, pDir)
	}

	return ips[0], nil
}
