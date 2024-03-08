package mapperclient

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=client.go -destination=mockclient/mocks.go
//go:generate go run github.com/Khan/genqlient@v0.7.0 ./genqlient.yaml
