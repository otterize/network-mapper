package cloudclient

//go:generate go run github.com/Khan/genqlient ./genqlient.yaml
//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -destination=./mocks/mocks.go  -package=cloudclientmocks -source=./cloud_client.go CloudClient
