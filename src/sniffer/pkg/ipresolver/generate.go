package ipresolver

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -destination=./mocks/mock_finder.go -package=ipresolvermocks --source=./ipresolver.go IpResolver
