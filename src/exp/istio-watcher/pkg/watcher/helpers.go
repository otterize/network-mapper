package istiowatcher

import (
	"fmt"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"time"
)

func ToGraphQLIstioConnections(connections map[*ConnectionWithPath]time.Time) []mapperclient.IstioConnection {
	connectionPairToConn := map[string]mapperclient.IstioConnection{}
	for connWithPath, timestamp := range connections {
		connectionPair := fmt.Sprintf("%s.%s", connWithPath.SourceWorkload, connWithPath.DestinationWorkload)
		istioConnection, ok := connectionPairToConn[connectionPair]
		if !ok {
			connectionPairToConn[connectionPair] = mapperclient.IstioConnection{
				SrcWorkload:          connWithPath.SourceWorkload,
				SrcWorkloadNamespace: connWithPath.SourceNamespace,
				DstWorkload:          connWithPath.DestinationWorkload,
				DstWorkloadNamespace: connWithPath.DestinationNamespace,
				RequestPaths:         []string{connWithPath.RequestPath},
				LastSeen:             timestamp,
			}
			continue
		}
		if slices.Contains(istioConnection.RequestPaths, connWithPath.RequestPath) {
			continue
		}
		// Reassign connection to map with newly appended request path
		istioConnection.RequestPaths = append(istioConnection.RequestPaths, connWithPath.RequestPath)

		if timestamp.After(istioConnection.LastSeen) {
			istioConnection.LastSeen = timestamp
		}
		connectionPairToConn[connectionPair] = istioConnection
	}

	return lo.Values(connectionPairToConn)
}
