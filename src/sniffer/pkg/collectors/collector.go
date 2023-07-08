package collectors

import (
	"github.com/otterize/network-mapper/src/sniffer/pkg/mapperclient"
	"github.com/sirupsen/logrus"
	"time"
)

type UniqueRequest struct {
	srcIP       string
	srcHostname string
	dest        string // IP or hostname
}

// For each unique request info, we store the time of the last request (no need to report duplicates)
type capturesMap map[UniqueRequest]time.Time

type NetworkCollector struct {
	capturedRequests capturesMap
}

func (c *NetworkCollector) resetData() {
	c.capturedRequests = make(capturesMap)
}

func (c *NetworkCollector) addCapturedRequest(srcIp string, srcHost string, dest string, seenAt time.Time) {
	req := UniqueRequest{srcIp, srcHost, dest}
	c.capturedRequests[req] = seenAt
}

func (c *NetworkCollector) CollectResults() []mapperclient.RecordedDestinationsForSrc {
	type srcInfo struct {
		Ip       string
		Hostname string
	}
	srcToDests := make(map[srcInfo][]mapperclient.Destination)

	for reqInfo, reqLastSeen := range c.capturedRequests {
		src := srcInfo{Ip: reqInfo.srcIP, Hostname: reqInfo.srcHostname}

		if _, ok := srcToDests[src]; !ok {
			srcToDests[src] = make([]mapperclient.Destination, 0)
		}
		srcToDests[src] = append(srcToDests[src], mapperclient.Destination{Destination: reqInfo.dest, LastSeen: reqLastSeen})
	}

	results := make([]mapperclient.RecordedDestinationsForSrc, 0)
	for src, destinations := range srcToDests {
		// Debug print the results
		logrus.Debugf("%s (%s):\n", src.Ip, src.Hostname)
		for _, dest := range destinations {
			logrus.Debugf("\t%s, %s\n", dest.Destination, dest.LastSeen)
		}

		results = append(results, mapperclient.RecordedDestinationsForSrc{SrcIp: src.Ip, SrcHostname: src.Hostname, Destinations: destinations})
	}

	c.resetData()

	return results
}
