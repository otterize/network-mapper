package utils

import (
	"errors"
	"fmt"
	"github.com/mpvl/unique"
	"github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/otterize/network-mapper/src/sniffer/pkg/config"
	"github.com/spf13/viper"
)

type ProcessScanCallback func(pid int64, pDir string)
type ProcessScanner func(callback ProcessScanCallback) error

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

func ExtractProcessHostname(pDir string) (string, error) {
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

func ExtractProcessIPAddr(pDir string) (string, error) {
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

func ExtractParentID(pDir string) (string, error) {
	contentBytes, err := os.ReadFile(fmt.Sprintf("%s/status", pDir))
	if err != nil {
		return "", err
	}

	content := string(contentBytes)
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if fields[0] == "PPid:" {
			return fields[1], nil
		}
	}
	return "", errors.New("can't find ppid")
}
