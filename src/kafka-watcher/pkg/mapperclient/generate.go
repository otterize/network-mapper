package mapperclient

//go:generate go run go.uber.org/mock/mockgen@v0.2.0 -source=client.go -destination=mockclient/mocks.go
//go:generate go run github.com/Khan/genqlient@v0.7.0 ./genqlient.yaml
