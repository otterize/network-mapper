package istiowatcher

import (
	"github.com/otterize/network-mapper/src/exp/istio-watcher/mapperclient"
	"github.com/samber/lo"
	"net/http"
	"time"
)

var HTTPMethodsToGQLMethods = map[string]mapperclient.HttpMethod{
	http.MethodGet:     mapperclient.HttpMethodGet,
	http.MethodPost:    mapperclient.HttpMethodPost,
	http.MethodPut:     mapperclient.HttpMethodPut,
	http.MethodDelete:  mapperclient.HttpMethodDelete,
	http.MethodOptions: mapperclient.HttpMethodOptions,
	http.MethodTrace:   mapperclient.HttpMethodTrace,
	http.MethodPatch:   mapperclient.HttpMethodPatch,
	http.MethodConnect: mapperclient.HttpMethodConnect,
}

type ConnectionPairWithPath struct {
	SourceWorkload      string `json:"sourceWorkload"`
	DestinationWorkload string `json:"destinationWorkload"`
	RequestPath         string `json:"requestPath"`
}

func ToGraphQLIstioConnections(connections map[ConnectionWithPath]time.Time) []mapperclient.IstioConnection {
	connectionPairToGraphQLConnection := map[ConnectionPairWithPath]mapperclient.IstioConnection{}

	for connWithPath, timestamp := range connections {
		connectionPair := ConnectionPairWithPath{
			SourceWorkload:      connWithPath.SourceWorkload,
			DestinationWorkload: connWithPath.DestinationWorkload,
			RequestPath:         connWithPath.RequestPath,
		}

		istioConnection, ok := connectionPairToGraphQLConnection[connectionPair]
		if !ok {
			istioConnection = mapperclient.IstioConnection{
				SrcWorkload:          connWithPath.SourceWorkload,
				SrcWorkloadNamespace: connWithPath.SourceNamespace,
				DstWorkload:          connWithPath.DestinationWorkload,
				DstWorkloadNamespace: connWithPath.DestinationNamespace,
				Path:                 connWithPath.RequestPath,
				LastSeen:             timestamp,
				Methods:              []mapperclient.HttpMethod{},
			}

			method, ok := HTTPMethodsToGQLMethods[connWithPath.RequestMethod]
			if ok {
				istioConnection.Methods = []mapperclient.HttpMethod{method}
			}

			connectionPairToGraphQLConnection[connectionPair] = istioConnection
			continue
		}

		if timestamp.After(istioConnection.LastSeen) {
			istioConnection.LastSeen = timestamp
		}

		method, ok := HTTPMethodsToGQLMethods[connWithPath.RequestMethod]
		if ok {
			istioConnection.Methods = append(istioConnection.Methods, method)
		}

		connectionPairToGraphQLConnection[connectionPair] = istioConnection
	}

	return lo.Values(connectionPairToGraphQLConnection)
}
