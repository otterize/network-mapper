package istiowatcher

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"net/http"
	"time"
)

var HTTPMethodsToGQLMethods = map[string]model.HTTPMethod{
	http.MethodGet:     model.HTTPMethodGet,
	http.MethodPost:    model.HTTPMethodPost,
	http.MethodPut:     model.HTTPMethodPut,
	http.MethodDelete:  model.HTTPMethodDelete,
	http.MethodOptions: model.HTTPMethodOptions,
	http.MethodTrace:   model.HTTPMethodTrace,
	http.MethodPatch:   model.HTTPMethodPatch,
	http.MethodConnect: model.HTTPMethodConnect,
}

type ConnectionPairWithPath struct {
	SourceWorkload      string `json:"sourceWorkload"`
	DestinationWorkload string `json:"destinationWorkload"`
	RequestPath         string `json:"requestPath"`
}

func ToGraphQLIstioConnections(connections map[ConnectionWithPath]time.Time) []model.IstioConnection {
	connectionPairToGraphQLConnection := map[ConnectionPairWithPath]model.IstioConnection{}

	for connWithPath, timestamp := range connections {
		connectionPair := ConnectionPairWithPath{
			SourceWorkload:      connWithPath.SourceWorkload,
			DestinationWorkload: connWithPath.DestinationWorkload,
			RequestPath:         connWithPath.RequestPath,
		}

		istioConnection, ok := connectionPairToGraphQLConnection[connectionPair]
		if !ok {
			istioConnection = model.IstioConnection{
				SrcWorkload:          connWithPath.SourceWorkload,
				SrcWorkloadNamespace: connWithPath.SourceNamespace,
				DstWorkload:          connWithPath.DestinationWorkload,
				DstWorkloadNamespace: connWithPath.DestinationNamespace,
				DstServiceName:       connWithPath.DestinationServiceName,
				Path:                 connWithPath.RequestPath,
				LastSeen:             timestamp,
			}

			method, ok := HTTPMethodsToGQLMethods[connWithPath.RequestMethod]
			if ok {
				istioConnection.Methods = []model.HTTPMethod{method}
			}

			connectionPairToGraphQLConnection[connectionPair] = istioConnection
			continue
		}

		if timestamp.After(istioConnection.LastSeen) {
			istioConnection.LastSeen = timestamp
		}

		method, ok := HTTPMethodsToGQLMethods[connWithPath.RequestMethod]
		if ok && !slices.Contains(istioConnection.Methods, method) {
			istioConnection.Methods = append(istioConnection.Methods, method)
		}

		connectionPairToGraphQLConnection[connectionPair] = istioConnection
	}

	return lo.Values(connectionPairToGraphQLConnection)
}
