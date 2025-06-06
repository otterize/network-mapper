package collectors

import (
	"github.com/otterize/network-mapper/src/mapperclient"
	"github.com/otterize/nilable"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"time"
)

type UniqueRequest struct {
	srcIP            string
	srcHostname      string
	destHostnameOrIP string // IP or hostname
	destIP           string
	destPort         nilable.Nilable[int]
}

type TimeAndTTL struct {
	lastSeen time.Time
	ttl      nilable.Nilable[int]
	srcPorts *SourcePortsSet
}

// For each unique request info, we store the time of the last request (no need to report duplicates) and last seen TTL.
type capturesMap map[UniqueRequest]TimeAndTTL

type SourcePortsSet map[int]struct{}

type NetworkCollector struct {
	capturedRequests capturesMap
}

func (c *NetworkCollector) resetData() {
	c.capturedRequests = make(capturesMap)
}

func (c *NetworkCollector) addCapturedRequest(srcIp string, srcHost string, destNameOrIP string, destIP string, seenAt time.Time, ttl nilable.Nilable[int], destPort *int, srcPort *int) {
	req := UniqueRequest{srcIp, srcHost, destNameOrIP, destIP, nilable.FromPtr(destPort)}
	existingRequest, requestFound := c.capturedRequests[req]
	if requestFound {
		existingSet := existingRequest.srcPorts
		if srcPort != nil {
			(*existingSet)[*srcPort] = struct{}{}
		}
		c.capturedRequests[req] = TimeAndTTL{seenAt, ttl, existingSet}
		return
	}

	newSet := make(SourcePortsSet)

	// if we reach here - the request is not found
	if srcPort != nil {
		newSet[*srcPort] = struct{}{}
	}

	c.capturedRequests[req] = TimeAndTTL{seenAt, ttl, lo.ToPtr(newSet)}
}

func (c *NetworkCollector) CollectResults() []mapperclient.RecordedDestinationsForSrc {
	type srcInfo struct {
		Ip       string
		Hostname string
	}
	srcToDests := make(map[srcInfo][]mapperclient.Destination)

	for reqInfo, timeAndTTL := range c.capturedRequests {
		src := srcInfo{Ip: reqInfo.srcIP, Hostname: reqInfo.srcHostname}

		if _, ok := srcToDests[src]; !ok {
			srcToDests[src] = make([]mapperclient.Destination, 0)
		}

		destination := mapperclient.Destination{
			Destination:     reqInfo.destHostnameOrIP,
			DestinationIP:   nilable.From(reqInfo.destIP),
			DestinationPort: reqInfo.destPort,
			LastSeen:        timeAndTTL.lastSeen,
			TTL:             timeAndTTL.ttl,
			SrcPorts:        lo.Keys(*timeAndTTL.srcPorts),
		}
		srcToDests[src] = append(srcToDests[src], destination)
	}

	results := make([]mapperclient.RecordedDestinationsForSrc, 0)
	for src, destinations := range srcToDests {
		// Debug print the results
		logrus.Debugf("%s (%s):\n", src.Ip, src.Hostname)
		for _, dest := range destinations {
			logrus.Debugf("    %s, %s", dest.Destination, dest.LastSeen)
		}

		results = append(results, mapperclient.RecordedDestinationsForSrc{SrcIp: src.Ip, SrcHostname: src.Hostname, Destinations: destinations})
	}

	c.resetData()

	return results
}
