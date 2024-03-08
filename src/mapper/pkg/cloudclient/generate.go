package cloudclient

import _ "github.com/suessflorian/gqlfetch"

// The check for $CI makes sure we don't redownload the schema in CI.
//go:generate sh -c "if [ -z $CI ]; then go run github.com/suessflorian/gqlfetch/gqlfetch --endpoint https://app.staging.otterize.com/api/graphql/v1beta > schema.graphql; fi"
//go:generate go run github.com/Khan/genqlient ./genqlient.yaml
//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -destination=./mocks/mocks.go  -package=cloudclientmocks -source=./cloud_client.go CloudClient
