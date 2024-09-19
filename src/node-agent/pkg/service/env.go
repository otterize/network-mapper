package service

import (
	"github.com/sirupsen/logrus"
	"os"
)

var (
	podName           string
	podNamespace      string
	nodeName          string
	printHttpRequests = false
)

func init() {
	podName = os.Getenv("POD_NAME")

	if podName == "" {
		logrus.Panic("POD_NAME environment variable must be set")
	}

	podNamespace = os.Getenv("POD_NAMESPACE")

	if podNamespace == "" {
		logrus.Panic("POD_NAMESPACE environment variable must be set")
	}

	nodeName = os.Getenv("NODE_NAME")

	if nodeName == "" {
		logrus.Panic("NODE_NAME environment variable must be set")
	}

	printHttpRequests = os.Getenv("OTTERIZE_PRINT_HTTP_REQUESTS") == "true"
}

func PodName() string {
	return podName
}

func PodNamespace() string {
	return podNamespace
}

func NodeName() string {
	return nodeName
}

func PrintHttpRequests() bool {
	return printHttpRequests
}
