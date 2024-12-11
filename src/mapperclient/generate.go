package mapperclient

//go:generate go run go.uber.org/mock/mockgen -source=client.go -destination=mockclient/mocks.go
//go:generate go run github.com/Khan/genqlient ./genqlient.yaml
