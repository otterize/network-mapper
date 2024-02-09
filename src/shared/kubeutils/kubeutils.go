package kubeutils

import (
	"github.com/otterize/intents-operator/src/shared/errors"
	"os"
	"strings"
)

const (
	DefaultClusterDomain = "cluster.local"
	namespaceFile        = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	resolvFile           = "/etc/resolv.conf"
)

func GetCurrentNamespace() (string, error) {
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return "", errors.Wrap(err)
	}
	return strings.TrimSpace(string(data)), nil
}

func GetClusterDomain() (string, error) {
	namespace, err := GetCurrentNamespace()
	if err != nil {
		return "", errors.Wrap(err)
	}
	data, err := os.ReadFile(resolvFile)
	if err != nil {
		return "", errors.Wrap(err)
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
	return "", errors.Errorf("could not deduce cluster domain from %s", resolvFile)
}
