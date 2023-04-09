package istiowatcher

import (
	"fmt"
	"github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"time"
)

func ToGraphQLIstioConnections(connections map[ConnectionWithPath]time.Time) []mapperclient.IstioConnection {
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
		if timestamp.After(istioConnection.LastSeen) {
			istioConnection.LastSeen = timestamp
		}
		
		if !slices.Contains(istioConnection.RequestPaths, connWithPath.RequestPath) {
			istioConnection.RequestPaths = append(istioConnection.RequestPaths, connWithPath.RequestPath)
		}

		connectionPairToConn[connectionPair] = istioConnection
	}

	return lo.Values(connectionPairToConn)
}
