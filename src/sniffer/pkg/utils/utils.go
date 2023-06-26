package utils

import (
	"fmt"
	"os"
	"strconv"

	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/spf13/viper"
)

type ProcessScanCallback func(pid int64, pDir string)

func ScanProcDirProcesses(callback ProcessScanCallback) error {
	hostProcDir := viper.GetString(config.HostProcDirKey)
	files, err := os.ReadDir(hostProcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		pid, err := strconv.ParseInt(f.Name(), 10, 64)
		if err != nil {
			// name is not a number, meaning it's not a process dir, skip
			continue
		}
		callback(pid, fmt.Sprintf("%s/%s", hostProcDir, f.Name()))
	}
	return nil
}
