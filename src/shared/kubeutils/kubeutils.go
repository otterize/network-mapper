package kubeutils

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultClusterDomain = "cluster.local"
	namespaceFile        = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	resolvFile           = "/etc/resolv.conf"
)

func GetClusterDomain() (string, error) {
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return "", err
	}
	namespace := strings.TrimSpace(string(data))

	data, err = os.ReadFile(resolvFile)
	if err != nil {
		return "", err
	}
	expectedSearchDomainPrefix := namespace + ".svc."
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		words := strings.Split(line, " ")
		if len(words) == 0 || words[0] != "search" {
			continue
		}
		for _, searchDomain := range words {
			if strings.HasPrefix(searchDomain, expectedSearchDomainPrefix) {
				return searchDomain[len(expectedSearchDomainPrefix):], nil
			}
		}
	}
	return "", fmt.Errorf("could not deduce cluster domain from %s", resolvFile)
}

var clusterDomain = ""

// GetClusterDomainOrDefault returns the k8s cluster domain of the cluster we are running in. when cannot resolve, returns
// the default cluster domain of k8s.
func GetClusterDomainOrDefault() string {
	if clusterDomain != "" {
		return clusterDomain
	}
	resolvedClusterDomain, err := GetClusterDomain()
	if err != nil {
		clusterDomain = DefaultClusterDomain
	} else {
		clusterDomain = resolvedClusterDomain
	}
	return clusterDomain
}
